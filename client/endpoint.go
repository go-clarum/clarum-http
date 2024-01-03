package client

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/goclarum/clarum/core/config"
	"github.com/goclarum/clarum/core/control"
	clarumstrings "github.com/goclarum/clarum/core/validators/strings"
	"github.com/goclarum/clarum/http/constants"
	"github.com/goclarum/clarum/http/internal/utils"
	"github.com/goclarum/clarum/http/internal/validators"
	"github.com/goclarum/clarum/http/message"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Endpoint struct {
	name            string
	baseUrl         string
	contentType     string
	client          *http.Client
	responseChannel chan *responsePair
}

type responsePair struct {
	response *http.Response
	error    error
}

func NewEndpoint(name string, baseUrl string, contentType string, timeout time.Duration) *Endpoint {
	client := http.Client{
		Timeout: getTimeoutWithDefault(timeout),
	}

	return &Endpoint{
		name:            name,
		baseUrl:         baseUrl,
		contentType:     contentType,
		client:          &client,
		responseChannel: make(chan *responsePair),
	}
}

func (endpoint *Endpoint) send(message *message.RequestMessage) error {
	logPrefix := clientLogPrefix(endpoint.name)

	if message == nil {
		return handleError("%s: message to send is nil", logPrefix)
	}

	slog.Debug(fmt.Sprintf("%s: message to send %s", logPrefix, message.ToString()))

	messageToSend := endpoint.getMessageToSend(message)
	slog.Debug(fmt.Sprintf("%s: will send message %s", logPrefix, messageToSend.ToString()))

	if err := validateMessageToSend(logPrefix, messageToSend); err != nil {
		return err
	}

	req, err := buildRequest(endpoint.name, messageToSend)
	// we return error here directly and not in the goroutine below
	// this way we can signal to the test synchronously that there was an error
	if err != nil {
		return handleError("%s: canceled message - %s", logPrefix, err)
	}

	go func() {
		control.RunningActions.Add(1)
		defer control.RunningActions.Done()

		logOutgoingRequest(logPrefix, message.MessagePayload, req)
		res, err := endpoint.client.Do(req)

		// we log the error here directly, but will do error handling downstream
		if err != nil {
			slog.Error(fmt.Sprintf("%s: error on response - %s", logPrefix, err))
		} else {
			logIncomingResponse(logPrefix, res)
		}

		responsePair := &responsePair{
			response: res,
			error:    err,
		}

		select {
		// we send the error downstream for it to be returned when an action is called
		case endpoint.responseChannel <- responsePair:
		case <-time.After(config.ActionTimeout()):
			handleError("%s: action timed out - no client receive action called in test", logPrefix)
		}
	}()

	return nil
}

// validationOptions pass by value is intentional
func (endpoint *Endpoint) receive(message *message.ResponseMessage, validationOptions receiveOptions) (*http.Response, error) {
	logPrefix := clientLogPrefix(endpoint.name)
	slog.Debug(fmt.Sprintf("%s: message to receive %s", logPrefix, message.ToString()))

	select {
	case responsePair := <-endpoint.responseChannel:
		if responsePair.error != nil {
			return responsePair.response, handleError("%s: error while receiving response - %s", logPrefix, responsePair.error)
		}

		messageToReceive := endpoint.getMessageToReceive(message)
		slog.Debug(fmt.Sprintf("%s: validating message %s", logPrefix, messageToReceive.ToString()))

		return responsePair.response, errors.Join(
			validators.ValidateHttpStatusCode(logPrefix, messageToReceive, responsePair.response.StatusCode),
			validators.ValidateHttpHeaders(logPrefix, &messageToReceive.Message, responsePair.response.Header),
			validators.ValidateHttpPayload(logPrefix, &messageToReceive.Message, responsePair.response.Body,
				validationOptions.expectedPayloadType))
	case <-time.After(config.ActionTimeout()):
		return nil, handleError("%s: receive action timed out - no response received for validation", logPrefix)
	}
}

// Put missing data into a message to send: baseUrl & ContentType Header
func (endpoint *Endpoint) getMessageToSend(message *message.RequestMessage) *message.RequestMessage {
	messageToSend := message.Clone()

	if clarumstrings.IsBlank(messageToSend.Url) {
		messageToSend.Url = endpoint.baseUrl
	}
	if len(messageToSend.Headers) == 0 || clarumstrings.IsBlank(messageToSend.Headers[constants.ContentTypeHeaderName]) {
		messageToSend.ContentType(endpoint.contentType)
	}

	return messageToSend
}

// Put missing data into message to receive: ContentType Header
func (endpoint *Endpoint) getMessageToReceive(message *message.ResponseMessage) *message.ResponseMessage {
	finalMessage := message.Clone()

	if clarumstrings.IsNotBlank(endpoint.contentType) {
		if len(finalMessage.Headers) == 0 {
			finalMessage.ContentType(endpoint.contentType)
		} else if _, exists := finalMessage.Headers[constants.ContentTypeHeaderName]; !exists {
			finalMessage.ContentType(endpoint.contentType)
		}
	}

	return finalMessage
}

func validateMessageToSend(prefix string, messageToSend *message.RequestMessage) error {
	if clarumstrings.IsBlank(messageToSend.Method) {
		return handleError("%s: message to send is invalid - missing HTTP method", prefix)
	}
	if clarumstrings.IsBlank(messageToSend.Url) {
		return handleError("%s: message to send is invalid - missing url", prefix)
	}
	if !utils.IsValidUrl(messageToSend.Url) {
		return handleError("%s: message to send is invalid - invalid url", prefix)
	}

	return nil
}

func buildRequest(prefix string, message *message.RequestMessage) (*http.Request, error) {
	url := utils.BuildPath(message.Url, message.Path)

	req, err := http.NewRequest(message.Method, url, bytes.NewBufferString(message.MessagePayload))
	if err != nil {
		slog.Error(fmt.Sprintf("%s: error - %s", prefix, err))
		return nil, err
	}

	for header, value := range message.Headers {
		req.Header.Set(header, value)
	}

	qParams := req.URL.Query()
	for key, values := range message.QueryParams {
		for _, value := range values {
			qParams.Add(key, value)
		}
	}
	req.URL.RawQuery = qParams.Encode()

	return req, nil
}

func handleError(format string, a ...any) error {
	errorMessage := fmt.Sprintf(format, a...)
	slog.Error(errorMessage)
	return errors.New(errorMessage)
}

func getTimeoutWithDefault(timeout time.Duration) time.Duration {
	var timeoutToSet time.Duration
	if timeout > 0 {
		timeoutToSet = timeout
	} else {
		timeoutToSet = 10 * time.Second
	}
	return timeoutToSet
}

func logOutgoingRequest(prefix string, payload string, req *http.Request) {
	slog.Info(fmt.Sprintf("%s: sending request ["+
		"method: %s, "+
		"url: %s, "+
		"headers: %s, "+
		"payload: %s"+
		"]",
		prefix, req.Method, req.URL, req.Header, payload))
}

// we read the body 'as is' for logging, after which we put it back into the response
// with an open reader so that it can be read downstream again
func logIncomingResponse(prefix string, res *http.Response) {
	bodyBytes, _ := io.ReadAll(res.Body)
	bodyString := ""

	err := res.Body.Close()
	if err != nil {
		slog.Error(fmt.Sprintf("%s: could not read response body - %s", prefix, err))
	} else {
		res.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		bodyString = string(bodyBytes)
	}

	slog.Info(fmt.Sprintf("%s: received response ["+
		"status: %s, "+
		"headers: %s, "+
		"payload: %s"+
		"]",
		prefix, res.Status, res.Header, bodyString))
}

func clientLogPrefix(endpointName string) string {
	return fmt.Sprintf("HTTP client %s", endpointName)
}
