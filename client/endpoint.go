package client

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/goclarum/clarum/core/config"
	"github.com/goclarum/clarum/core/control"
	"github.com/goclarum/clarum/core/durations"
	"github.com/goclarum/clarum/core/logging"
	clarumstrings "github.com/goclarum/clarum/core/validators/strings"
	"github.com/goclarum/clarum/http/constants"
	"github.com/goclarum/clarum/http/internal/utils"
	"github.com/goclarum/clarum/http/internal/validators"
	"github.com/goclarum/clarum/http/message"
	"io"
	"net/http"
	"time"
)

type Endpoint struct {
	name            string
	baseUrl         string
	contentType     string
	client          *http.Client
	responseChannel chan *responsePair
	logger          *logging.Logger
}

type responsePair struct {
	response *http.Response
	error    error
}

// TODO: handle empty name
func newEndpoint(name string, baseUrl string, contentType string, timeout time.Duration) *Endpoint {
	client := http.Client{
		Timeout: durations.GetDurationWithDefault(timeout, 10*time.Second),
	}

	return &Endpoint{
		name:            name,
		baseUrl:         baseUrl,
		contentType:     contentType,
		client:          &client,
		responseChannel: make(chan *responsePair),
		logger:          logging.NewLogger(config.LoggingLevel(), clientLogPrefix(name)),
	}
}

func (endpoint *Endpoint) send(message *message.RequestMessage) error {
	if message == nil {
		return endpoint.handleError("message to send is nil", nil)
	}

	endpoint.logger.Debugf("message to send %s", message.ToString())

	messageToSend := endpoint.getMessageToSend(message)
	endpoint.logger.Debugf("will send message %s", messageToSend.ToString())

	if err := endpoint.validateMessageToSend(messageToSend); err != nil {
		return err
	}

	req, err := endpoint.buildRequest(messageToSend)
	// we return error here directly and not in the goroutine below
	// this way we can signal to the test synchronously that there was an error
	if err != nil {
		return endpoint.handleError("canceled message", err)
	}

	go func() {
		control.RunningActions.Add(1)
		defer control.RunningActions.Done()

		endpoint.logOutgoingRequest(message.MessagePayload, req)
		res, err := endpoint.client.Do(req)

		// we log the error here directly, but will do error handling downstream
		if err != nil {
			endpoint.logger.Errorf("error on response - %s", err)
		} else {
			endpoint.logIncomingResponse(res)
		}

		responsePair := &responsePair{
			response: res,
			error:    err,
		}

		select {
		// we send the error downstream for it to be returned when an action is called
		case endpoint.responseChannel <- responsePair:
		case <-time.After(config.ActionTimeout()):
			endpoint.handleError("action timed out - no client receive action called in test", nil)
		}
	}()

	return nil
}

// validationOptions pass by value is intentional
func (endpoint *Endpoint) receive(message *message.ResponseMessage, validationOptions receiveOptions) (*http.Response, error) {
	endpoint.logger.Debugf("message to receive %s", message.ToString())

	select {
	case responsePair := <-endpoint.responseChannel:
		if responsePair.error != nil {
			return responsePair.response, endpoint.handleError("error while receiving response", responsePair.error)
		}

		messageToReceive := endpoint.getMessageToReceive(message)
		endpoint.logger.Debugf("validating message %s", messageToReceive.ToString())

		return responsePair.response, errors.Join(
			validators.ValidateHttpStatusCode(messageToReceive, responsePair.response.StatusCode, endpoint.logger),
			validators.ValidateHttpHeaders(&messageToReceive.Message, responsePair.response.Header, endpoint.logger),
			validators.ValidateHttpPayload(&messageToReceive.Message, responsePair.response.Body,
				validationOptions.expectedPayloadType, endpoint.logger))
	case <-time.After(config.ActionTimeout()):
		return nil, endpoint.handleError("receive action timed out - no response received for validation", nil)
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

func (endpoint *Endpoint) validateMessageToSend(messageToSend *message.RequestMessage) error {
	if clarumstrings.IsBlank(messageToSend.Method) {
		return endpoint.handleError("message to send is invalid - missing HTTP method", nil)
	}
	if clarumstrings.IsBlank(messageToSend.Url) {
		return endpoint.handleError("message to send is invalid - missing url", nil)
	}
	if !utils.IsValidUrl(messageToSend.Url) {
		return endpoint.handleError("message to send is invalid - invalid url", nil)
	}

	return nil
}

func (endpoint *Endpoint) buildRequest(message *message.RequestMessage) (*http.Request, error) {
	url := utils.BuildPath(message.Url, message.Path)

	req, err := http.NewRequest(message.Method, url, bytes.NewBufferString(message.MessagePayload))
	if err != nil {
		endpoint.logger.Errorf("error - %s", err)
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

func (endpoint *Endpoint) logOutgoingRequest(payload string, req *http.Request) {
	endpoint.logger.Infof("sending HTTP request ["+
		"method: %s, "+
		"url: %s, "+
		"headers: %s, "+
		"payload: %s"+
		"]",
		req.Method, req.URL, req.Header, payload)
}

// we read the body 'as is' for logging, after which we put it back into the response
// with an open reader so that it can be read downstream again
func (endpoint *Endpoint) logIncomingResponse(res *http.Response) {
	bodyBytes, _ := io.ReadAll(res.Body)
	bodyString := ""

	err := res.Body.Close()
	if err != nil {
		endpoint.logger.Errorf("could not read response body - %s", err)
	} else {
		res.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		bodyString = string(bodyBytes)
	}

	endpoint.logger.Infof("received HTTP response ["+
		"status: %s, "+
		"headers: %s, "+
		"payload: %s"+
		"]",
		res.Status, res.Header, bodyString)
}

func clientLogPrefix(endpointName string) string {
	return fmt.Sprintf("%s: ", endpointName)
}
