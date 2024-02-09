package errors

import (
	"github.com/goclarum/clarum/http/message"
	"testing"
)

// The following tests check client send request validation errors.

func TestClientSendNilMessage(t *testing.T) {
	expectedErrors := []string{
		"errorsClient: message to send is nil",
	}

	e1 := errorsClient.Send().Message(nil)

	checkErrors(t, expectedErrors, e1)
}

func TestClientSendNilUrl(t *testing.T) {
	expectedErrors := []string{
		"errorsClient: message to send is invalid - missing url",
	}

	e1 := errorsClient.Send().Message(message.Get())

	checkErrors(t, expectedErrors, e1)
}

func TestClientSendInvalidUrl(t *testing.T) {
	expectedErrors := []string{
		"errorsClient: message to send is invalid - invalid url",
	}

	e1 := errorsClient.Send().Message(message.Get().BaseUrl("http:/localhost:8081"))
	e2 := errorsClient.Send().Message(message.Get().BaseUrl("som e thi ng"))

	checkErrors(t, expectedErrors, e1)
	checkErrors(t, expectedErrors, e2)
}

func TestClientSendInvalidMessageMethod(t *testing.T) {
	expectedErrors := []string{
		"errorsClient: message to send is invalid - missing HTTP method",
	}

	request := &message.RequestMessage{
		// Method: intentionally missing here
		Url: "something",
	}
	e1 := errorsClient.Send().Message(request)

	checkErrors(t, expectedErrors, e1)
}
