package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/go-clarum/clarum-core/config"
	"github.com/go-clarum/clarum-core/control"
	"github.com/go-clarum/clarum-core/logging"
	clarumstrings "github.com/go-clarum/clarum-core/validators/strings"
	"github.com/goclarum/clarum/http/constants"
	"github.com/goclarum/clarum/http/internal/validators"
	"github.com/goclarum/clarum/http/message"
	"io"
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
	logger                   *logging.Logger
}

type endpointContext struct {
	endpointName             string
	requestValidationChannel chan *http.Request
	sendChannel              chan *sendPair
	logger                   *logging.Logger
}

type sendPair struct {
	response *message.ResponseMessage
	error    error
}

func newServerEndpoint(name string, port uint, contentType string, timeout time.Duration) *Endpoint {
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
		logger:                   logging.NewLogger(config.LoggingLevel(), serverLogPrefix(name)),
	}

	// feature: start automatically = true/false; to simulate connection errors
	se.start(ctx, cancelCtx, timeout)

	return se
}

// this Method is blocking, until a request is received
func (endpoint *Endpoint) receive(message *message.RequestMessage, validationOptions receiveOptions) (*http.Request, error) {
	endpoint.logger.Debugf("message to receive %s", message.ToString())
	messageToReceive := endpoint.getMessageToReceive(message)

	select {
	case receivedRequest := <-endpoint.requestValidationChannel:
		endpoint.logger.Debugf("validation message %s", messageToReceive.ToString())

		return receivedRequest, errors.Join(
			validators.ValidatePath(messageToReceive, receivedRequest.URL, endpoint.logger),
			validators.ValidateHttpMethod(messageToReceive, receivedRequest.Method, endpoint.logger),
			validators.ValidateHttpHeaders(&messageToReceive.Message, receivedRequest.Header, endpoint.logger),
			validators.ValidateHttpQueryParams(messageToReceive, receivedRequest.URL, endpoint.logger),
			validators.ValidateHttpPayload(&messageToReceive.Message, receivedRequest.Body,
				validationOptions.expectedPayloadType, endpoint.logger))
	case <-time.After(config.ActionTimeout()):
		return nil, endpoint.handleError("receive action timed out - no request received for validation", nil)
	}
}

func (endpoint *Endpoint) send(message *message.ResponseMessage) error {
	messageToSend := endpoint.getMessageToSend(message)

	err := endpoint.validateMessageToSend(messageToSend)

	// we must always send a signal downstream so that the handler is not blocked
	toSend := &sendPair{
		response: messageToSend,
		error:    err,
	}

	select {
	case endpoint.sendChannel <- toSend:
		return err
	case <-time.After(config.ActionTimeout()):
		return endpoint.handleError("send action timed out - no request received for validation", nil)
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
				logger:                   endpoint.logger,
			}
			ctx = context.WithValue(ctx, contextNameKey, endpointContext)
			return ctx
		},
	}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			endpoint.logger.Errorf("error - %s", err)
		} else {
			endpoint.logger.Info("closed server")
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
	ctx := request.Context().Value(contextNameKey).(*endpointContext)
	defer finishOrRecover(ctx.logger)

	logIncomingRequest(ctx.logger, request)

	select {
	case ctx.requestValidationChannel <- request:
		ctx.logger.Debug("received request was sent to validation channel")
	case <-time.After(config.ActionTimeout()):
		ctx.logger.Warn("request handling timed out - no server receive action called in test")
	}

	select {
	case sendPair := <-ctx.sendChannel:
		// error from upstream - we send a response to close the HTTP cycle
		if sendPair.error != nil {
			sendDefaultErrorResponse(ctx.logger, "request handler received error from upstream", resWriter)
			return
		}

		// check if response is empty - we send a response to close the HTTP cycle
		if sendPair.response == nil {
			sendDefaultErrorResponse(ctx.logger, "request handler received empty ResponseMesage", resWriter)
			return
		}

		sendResponse(ctx.logger, sendPair, resWriter)
	case <-time.After(config.ActionTimeout()):
		ctx.logger.Warn("response handling timed out - no server send action called in test")
	}
}

func sendResponse(logger *logging.Logger, sendPair *sendPair, resWriter http.ResponseWriter) {
	for header, value := range sendPair.response.Headers {
		resWriter.Header().Set(header, value)
	}

	resWriter.WriteHeader(sendPair.response.StatusCode)

	_, err := io.WriteString(resWriter, sendPair.response.MessagePayload)
	if err != nil {
		logger.Errorf("could not write response body - %s", err)
	}
	logOutgoingResponse(logger, sendPair.response.StatusCode, sendPair.response.MessagePayload, resWriter)
}

func sendDefaultErrorResponse(logger *logging.Logger, errorMessage string, resWriter http.ResponseWriter) {
	logger.Error(errorMessage)
	resWriter.WriteHeader(http.StatusInternalServerError)
	logOutgoingResponse(logger, http.StatusInternalServerError, "", resWriter)
}

func (endpoint *Endpoint) validateMessageToSend(messageToSend *message.ResponseMessage) error {
	if messageToSend.StatusCode < 100 || messageToSend.StatusCode > 999 {
		return endpoint.handleError(fmt.Sprintf("message to send is invalid - unsupported status code [%d]",
			messageToSend.StatusCode), nil)
	}

	return nil
}

func (endpoint *Endpoint) handleError(message string, err error) error {
	var errorMessage string
	if err != nil {
		errorMessage = message + " - " + err.Error()
	} else {
		errorMessage = message
	}
	endpoint.logger.Errorf(errorMessage)
	return errors.New(endpoint.logger.Prefix() + errorMessage)
}

func finishOrRecover(logger *logging.Logger) {
	control.RunningActions.Done()

	if r := recover(); r != nil {
		logger.Errorf("endpoint panicked: error - %s", r)
	}
}

// we read the body 'as is' for logging, after which we put it back into the request
// with an open reader so that it can be read downstream again
func logIncomingRequest(logger *logging.Logger, request *http.Request) {
	bodyBytes, _ := io.ReadAll(request.Body)
	bodyString := ""

	err := request.Body.Close()
	if err != nil {
		logger.Errorf("could not read request body - %s", err)
	} else {
		request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		bodyString = string(bodyBytes)
	}

	logger.Infof("received HTTP request ["+
		"method: %s, "+
		"url: %s, "+
		"headers: %s, "+
		"payload: %s"+
		"]",
		request.Method, request.URL.String(), request.Header, bodyString)
}

func logOutgoingResponse(logger *logging.Logger, statusCode int, payload string, res http.ResponseWriter) {
	logger.Infof("sending response ["+
		"status: %d, "+
		"headers: %s, "+
		"payload: %s"+
		"]",
		statusCode, res.Header(), payload)
}

func serverLogPrefix(endpointName string) string {
	return fmt.Sprintf("%s: ", endpointName)
}
