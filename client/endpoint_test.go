package client

import (
	"github.com/goclarum/clarum/http/constants"
	"github.com/goclarum/clarum/http/message"
	"net/http"
	"testing"
	"time"
)

func TestNewEndpoint(t *testing.T) {
	endpoint := newEndpoint("newName", "myUrl", constants.ContentTypeJsonHeader,
		time.Second*2)

	if endpoint == nil {
		t.Errorf("endpoint must not be null")
	}
	if endpoint.name != "newName" {
		t.Errorf("invalid endpoint.name")
	}
	if endpoint.baseUrl != "myUrl" {
		t.Errorf("invalid endpoint.baseUrl")
	}
	if endpoint.contentType != constants.ContentTypeJsonHeader {
		t.Errorf("invalid endpoint.contentType")
	}
	if endpoint.client == nil {
		t.Errorf("endpoint.client must not be null")
	}
	if endpoint.client.Timeout != time.Second*2 {
		t.Errorf("invalid endpoint.client.Timeout")
	}
	if endpoint.responseChannel == nil {
		t.Errorf("endpoint.responseChannel must not be null")
	}
}

func TestNewEndpointDefaultTimeout(t *testing.T) {
	endpoint := newEndpoint("a", "b", "c", 0)

	if endpoint.client.Timeout != time.Second*10 {
		t.Errorf("invalid endpoint.client.Timeout")
	}
}

func TestGetMessageToSend(t *testing.T) {
	endpoint := newEndpoint("name", "endpointUrl", "endpointContent", 0)

	initialRequest := message.Get("my-url")
	finalRequest := endpoint.getMessageToSend(initialRequest)

	if initialRequest == finalRequest {
		t.Errorf("message has not been cloned.")
	}
	if initialRequest.Equals(finalRequest) {
		t.Errorf("messages must not be equal.")
	}
	if finalRequest.Url != "endpointUrl" {
		t.Errorf("invalid finalRequest.Url")
	}
	if finalRequest.Headers[constants.ContentTypeHeaderName] != "endpointContent" {
		t.Errorf("invalid finalRequest.ContentType")
	}
}

func TestGetMessageToSendNoChangeInFinalRequest(t *testing.T) {
	endpoint := newEndpoint("name", "endpointUrl", "endpointContent", 0)

	initialRequest := message.Get("my-url").
		BaseUrl("otherBaseUrl").
		ContentType("otherContentType")
	finalRequest := endpoint.getMessageToSend(initialRequest)

	if initialRequest == finalRequest {
		t.Errorf("message has not been cloned.")
	}
	if !initialRequest.Equals(finalRequest) {
		t.Errorf("messages must be equal.")
	}
	if finalRequest.Url != "otherBaseUrl" {
		t.Errorf("invalid finalRequest.Url")
	}
	if finalRequest.Headers[constants.ContentTypeHeaderName] != "otherContentType" {
		t.Errorf("invalid finalRequest.ContentType")
	}
}

func TestGetMessageToReceive(t *testing.T) {
	endpoint := newEndpoint("name", "endpointUrl", "endpointContent", 0)

	initialResponse := message.Response(http.StatusOK)
	finalResponse := endpoint.getMessageToReceive(initialResponse)

	if initialResponse == finalResponse {
		t.Errorf("message has not been cloned.")
	}
	if initialResponse.Equals(finalResponse) {
		t.Errorf("messages must not be equal.")
	}
	if finalResponse.Headers[constants.ContentTypeHeaderName] != "endpointContent" {
		t.Errorf("invalid finalResponse.ContentType")
	}
}

func TestGetMessageToReceiveNoChangeInFinalResponse(t *testing.T) {
	endpoint := newEndpoint("name", "endpointUrl", "endpointContent", 0)

	initialResponse := message.Response(http.StatusOK).
		ContentType("otherContentType")
	finalResponse := endpoint.getMessageToReceive(initialResponse)

	if initialResponse == finalResponse {
		t.Errorf("message has not been cloned.")
	}
	if !initialResponse.Equals(finalResponse) {
		t.Errorf("messages must be equal.")
	}
	if finalResponse.Headers[constants.ContentTypeHeaderName] != "otherContentType" {
		t.Errorf("invalid finalResponse.ContentType")
	}
}

func TestValidateMessageToSend(t *testing.T) {
	endpoint := newEndpoint("name", "baseUrl", "", 0)
	request := message.Get("my-url").
		BaseUrl("http://localhost:8080")

	if err := endpoint.validateMessageToSend(request); err != nil {
		t.Errorf("request must be valid")
	}

	request = message.Get("my-url")
	if err := endpoint.validateMessageToSend(request); !(err != nil && err.Error() == "name: message to send is invalid - missing url") {
		t.Errorf("invalid error")
	}

	request = message.Get("my-url").BaseUrl("something")
	if err := endpoint.validateMessageToSend(request); !(err != nil && err.Error() == "name: message to send is invalid - invalid url") {
		t.Errorf("invalid error")
	}

	request = &message.RequestMessage{}
	if err := endpoint.validateMessageToSend(request); !(err != nil && err.Error() == "name: message to send is invalid - missing HTTP method") {
		t.Errorf("invalid error")
	}
}

func TestBuildRequest(t *testing.T) {
	endpoint := newEndpoint("name", "baseUrl", "", 0)

	requestMessage := message.Post("my", "api/v0").
		BaseUrl("http://localhost:8080").
		ContentType("text/plain").
		QueryParam("someParameter", "someValue").
		Payload("batman!")

	newRequest, err := endpoint.buildRequest(requestMessage)
	if err != nil {
		t.Errorf("error is unexpected")
	}

	if newRequest.Method != http.MethodPost {
		t.Errorf("invalid newRequest.Method")
	}
	if newRequest.URL.Scheme != "http" {
		t.Errorf("invalid newRequest.URL.Scheme")
	}
	if newRequest.URL.Host != "localhost:8080" {
		t.Errorf("invalid newRequest.URL.Host")
	}
	if newRequest.URL.Path != "/my/api/v0" {
		t.Errorf("invalid newRequest.URL.Path")
	}

	actualHeaderValue := newRequest.Header[constants.ContentTypeHeaderName]
	if actualHeaderValue[0] != "text/plain" {
		t.Errorf("invalid newRequest.Headers[Content-Type]")
	}

	queryParamValues := newRequest.URL.Query()["someParameter"]
	if queryParamValues[0] != "someValue" {
		t.Errorf("invalid newRequest.URL.QueryParams[someParameter]")
	}
}
