package message

import (
	"github.com/go-clarum/clarum-http/constants"
	"net/http"
	"testing"
)

func TestHTTPVerbs(t *testing.T) {
	if Get().Method != http.MethodGet {
		t.Errorf("Expected %s.", http.MethodGet)
	}
	if Head().Method != http.MethodHead {
		t.Errorf("Expected %s.", http.MethodHead)
	}
	if Post().Method != http.MethodPost {
		t.Errorf("Expected %s.", http.MethodPost)
	}
	if Put().Method != http.MethodPut {
		t.Errorf("Expected %s.", http.MethodPut)
	}
	if Delete().Method != http.MethodDelete {
		t.Errorf("Expected %s.", http.MethodDelete)
	}
	if Options().Method != http.MethodOptions {
		t.Errorf("Expected %s.", http.MethodOptions)
	}
	if Trace().Method != http.MethodTrace {
		t.Errorf("Expected %s.", http.MethodTrace)
	}
	if Patch().Method != http.MethodPatch {
		t.Errorf("Expected %s.", http.MethodPatch)
	}
}

func TestRequestBuilder(t *testing.T) {
	actual := Post("my", "api/v0").
		BaseUrl("http//localhost:8080").
		ContentType("text/plain").
		Authorization("1232341").
		Payload("batman!")

	expected := RequestMessage{
		Method: http.MethodPost,
		Url:    "http//localhost:8080",
		Path:   "my/api/v0",
		Message: Message{
			MessagePayload: "batman!",
			Headers: map[string]string{
				constants.ContentTypeHeaderName:   "text/plain",
				constants.AuthorizationHeaderName: "1232341",
			},
		},
	}

	if !actual.Equals(&expected) {
		t.Errorf("Message is not as expected.")
	}
}

func TestRequestClone(t *testing.T) {
	message := Get("my-url").
		BaseUrl("http//localhost:8080").
		ContentType("text/plain").
		Authorization("1232341").
		Payload("my payload")

	clonedMessage := message.Clone()

	if clonedMessage == message {
		t.Errorf("Message has not been cloned.")
	}

	if !clonedMessage.Equals(message) {
		t.Errorf("Messages are not equal.")
	}
}
