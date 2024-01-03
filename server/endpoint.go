package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/goclarum/clarum/core/config"
	"github.com/goclarum/clarum/core/control"
	clarumstrings "github.com/goclarum/clarum/core/validators/strings"
	"github.com/goclarum/clarum/http/constants"
	"github.com/goclarum/clarum/http/internal/validators"
	"github.com/goclarum/clarum/http/message"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"
)

const contextNameKey = "endpointContext"

type Endpoint struct {
	name                     string
	port                     uint
	contentType              string
	server                   *http.Server
	context                  *context.Context
	requestValidationChannel chan *http.Request
	sendChannel              chan *sendPair
}

type endpointContext struct {
	endpointName             string
	requestValidationChannel chan *http.Request
	sendChannel              chan *sendPair
}

type sendPair struct {
	response *message.ResponseMessage
	error    error
}

func NewServerEndpoint(name string, port uint, contentType string, timeout time.Duration) *Endpoint {
	ctx, cancelCtx := context.WithCancel(context.Background())
	sendChannel := make(chan *sendPair)
	requestChannel := make(chan *http.Request)

	se := &Endpoint{
		name:                     name,
		port:                     port,
		contentType:              contentType,
		context:                  &ctx,
		sendChannel:              sendChannel,
		requestValidationChannel: requestChannel,
	}

	// feature: start automatically = true/false; to simulate connection errors
	se.start(ctx, cancelCtx, timeout)

	return se
}

// this Method is blocking, until a request is received
func (endpoint *Endpoint) receive(message *message.RequestMessage, validationOptions receiveOptions) (*http.Request, error) {
	logPrefix := serverLogPrefix(endpoint.name)
	slog.Debug(fmt.Sprintf("%s: message to receive %s", logPrefix, message.ToString()))
	messageToReceive := endpoint.getMessageToReceive(message)

	select {
	case receivedRequest := <-endpoint.requestValidationChannel:
		slog.Debug(fmt.Sprintf("%s: validation message %s", logPrefix, messageToReceive.ToString()))

		return receivedRequest, errors.Join(
			validators.ValidatePath(logPrefix, messageToReceive, receivedRequest.URL),
			validators.ValidateHttpMethod(logPrefix, messageToReceive, receivedRequest.Method),
			validators.ValidateHttpHeaders(logPrefix, &messageToReceive.Message, receivedRequest.Header),
			validators.ValidateHttpQueryParams(logPrefix, messageToReceive, receivedRequest.URL),
			validators.ValidateHttpPayload(logPrefix, &messageToReceive.Message, receivedRequest.Body,
				validationOptions.expectedPayloadType))
	case <-time.After(config.ActionTimeout()):
		return nil, handleError("%s: receive action timed out - no request received for validation", logPrefix)
	}
}

func (endpoint *Endpoint) send(message *message.ResponseMessage) error {
	logPrefix := serverLogPrefix(endpoint.name)
	messageToSend := endpoint.getMessageToSend(message)

	err := validateMessageToSend(logPrefix, messageToSend)

	// we must always send a signal downstream so that the handler is not blocked
	toSend := &sendPair{
		response: messageToSend,
		error:    err,
	}

	select {
	case endpoint.sendChannel <- toSend:
		return err
	case <-time.After(config.ActionTimeout()):
		return handleError("%s: send action timed out - no request received for validation", logPrefix)
	}
}

func (endpoint *Endpoint) getMessageToReceive(message *message.RequestMessage) *message.RequestMessage {
	finalMessage := message.Clone()

	if clarumstrings.IsNotBlank(endpoint.contentType) {
		if len(finalMessage.Headers) == 0 {
			finalMessage.ContentType(endpoint.contentType)
		} else if _, exists := finalMessage.Headers[constants.ContentTypeHeaderName]; exists {
			finalMessage.ContentType(endpoint.contentType)
		}
	}

	return finalMessage
}

// we clone the message, so that further interaction with it in the test will not have any side effects
func (endpoint *Endpoint) getMessageToSend(message *message.ResponseMessage) *message.ResponseMessage {
	finalMessage := message.Clone()

	if len(finalMessage.Headers) == 0 || len(finalMessage.Headers[constants.ContentTypeHeaderName]) == 0 {
		finalMessage.ContentType(endpoint.contentType)
	}

	return finalMessage
}

func (endpoint *Endpoint) start(ctx context.Context, cancelCtx context.CancelFunc, timeout time.Duration) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", requestHandler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", endpoint.port),
		Handler:      mux,
		WriteTimeout: timeout,
		BaseContext: func(l net.Listener) context.Context {
			endpointContext := &endpointContext{
				endpointName:             endpoint.name,
				requestValidationChannel: endpoint.requestValidationChannel,
				sendChannel:              endpoint.sendChannel,
			}
			ctx = context.WithValue(ctx, contextNameKey, endpointContext)
			return ctx
		},
	}

	go func() {
		logPrefix := serverLogPrefix(endpoint.name)
		if err := server.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				fmt.Println(fmt.Sprintf("%s: closed", logPrefix))
			} else {
				fmt.Println(fmt.Sprintf("%s: error - %s", logPrefix, err))
			}
		} else {
			fmt.Println(fmt.Sprintf("%s: closed - %s", logPrefix, err))
		}

		cancelCtx()
	}()

	endpoint.server = server
}

// The requestHandler is started when the server receives a request.
// The request is sent to the requestValidationChannel to be picked up by a test action (validation).
// After sending the request to the channel, the handler is blocked until the send() test action
// provides a response message. This way we can control, inside the test, when a response will be sent.
// The handler blocks until a timeout is triggered
func requestHandler(resWriter http.ResponseWriter, request *http.Request) {
	control.RunningActions.Add(1)
	defer finishOrRecover()

	ctx := request.Context().Value(contextNameKey).(*endpointContext)

	logPrefix := serverLogPrefix(ctx.endpointName)
	logIncomingRequest(logPrefix, request)

	select {
	case ctx.requestValidationChannel <- request:
		slog.Debug(fmt.Sprintf("%s: received HTTP request sent to request validation channel", logPrefix))
	case <-time.After(config.ActionTimeout()):
		slog.Warn(fmt.Sprintf("%s: request handling timed out - no server receive action called in test", logPrefix))
	}

	select {
	case sendPair := <-ctx.sendChannel:
		// error from upstream - we send a response to close the HTTP cycle
		if sendPair.error != nil {
			errorMessage := fmt.Sprintf("%s: request handler received error from upstream", logPrefix)
			sendDefaultErrorResponse(logPrefix, errorMessage, resWriter)
			return
		}

		// check if response is empty - we send a response to close the HTTP cycle
		if sendPair.response == nil {
			errorMessage := fmt.Sprintf("%s: request handler received empty ResponseMesage", logPrefix)
			sendDefaultErrorResponse(logPrefix, errorMessage, resWriter)
			return
		}

		sendResponse(logPrefix, sendPair, resWriter)
	case <-time.After(config.ActionTimeout()):
		slog.Warn(fmt.Sprintf("%s: response handling timed out - no server send action called in test", logPrefix))
	}
}

func sendResponse(logPrefix string, sendPair *sendPair, resWriter http.ResponseWriter) {
	for header, value := range sendPair.response.Headers {
		resWriter.Header().Set(header, value)
	}

	resWriter.WriteHeader(sendPair.response.StatusCode)

	_, err := io.WriteString(resWriter, sendPair.response.MessagePayload)
	if err != nil {
		slog.Error(fmt.Sprintf("%s: could not write response body - %s", logPrefix, err))
	}
	logOutgoingResponse(logPrefix, sendPair.response.StatusCode, sendPair.response.MessagePayload, resWriter)
}

func sendDefaultErrorResponse(logPrefix string, errorMessage string, resWriter http.ResponseWriter) {
	slog.Error(errorMessage)
	resWriter.WriteHeader(http.StatusInternalServerError)
	logOutgoingResponse(logPrefix, http.StatusInternalServerError, "", resWriter)
}

func validateMessageToSend(prefix string, messageToSend *message.ResponseMessage) error {
	if messageToSend.StatusCode < 100 || messageToSend.StatusCode > 999 {
		return handleError("%s: message to send is invalid - unsupported status code [%d]",
			prefix, messageToSend.StatusCode)
	}

	return nil
}

func handleError(format string, a ...any) error {
	errorMessage := fmt.Sprintf(format, a...)
	slog.Error(errorMessage)
	return errors.New(errorMessage)
}

func finishOrRecover() {
	control.RunningActions.Done()

	if r := recover(); r != nil {
		slog.Error(fmt.Sprintf("HTTP server endpoint panicked: error - %s", r))
	}
}

// we read the body 'as is' for logging, after which we put it back into the request
// with an open reader so that it can be read downstream again
func logIncomingRequest(logPrefix string, request *http.Request) {
	bodyBytes, _ := io.ReadAll(request.Body)
	bodyString := ""

	err := request.Body.Close()
	if err != nil {
		slog.Error(fmt.Sprintf("%s: could not read request body - %s", logPrefix, err))
	} else {
		request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		bodyString = string(bodyBytes)
	}

	slog.Info(fmt.Sprintf("%s: received request ["+
		"method: %s, "+
		"url: %s, "+
		"headers: %s, "+
		"payload: %s"+
		"]",
		logPrefix, request.Method, request.URL.String(), request.Header, bodyString))
}

func logOutgoingResponse(prefix string, statusCode int, payload string, res http.ResponseWriter) {
	slog.Info(fmt.Sprintf("%s: sending response ["+
		"status: %d, "+
		"headers: %s, "+
		"payload: %s"+
		"]",
		prefix, statusCode, res.Header(), payload))
}

func serverLogPrefix(endpointName string) string {
	return fmt.Sprintf("HTTP server %s", endpointName)
}
