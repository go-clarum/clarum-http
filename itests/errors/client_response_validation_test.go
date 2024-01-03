package errors

import (
	"github.com/goclarum/clarum/http/message"
	"net/http"
	"testing"
)

// The following tests check client receive response validation errors.

// HTTP header validation error: header missing
func TestHeaderMissingResponseValidation(t *testing.T) {
	expectedErrors := []string{
		"HTTP client errorsClient: validation error - header <etag> missing",
	}

	e1 := errorsClient.Send().
		Message(message.Get().BaseUrl("http://localhost:8083"))

	_, e2 := errorsServer.Receive().Message(message.Get())
	e3 := errorsServer.Send().
		Message(message.Response(http.StatusOK))

	_, e4 := errorsClient.Receive().
		Message(message.Response(http.StatusOK).
			ETag("132r1r312e1"))

	checkErrors(t, expectedErrors, e1, e2, e3, e4)
}

// HTTP header validation error: header value incorrect
func TestHeaderInvalidResponseValidation(t *testing.T) {
	expectedErrors := []string{
		"HTTP client errorsClient: validation error - header <someheader> mismatch - expected [wrongValue] but received [[someValue]]",
	}

	e1 := errorsClient.Send().
		Message(message.Get().BaseUrl("http://localhost:8083"))

	_, e2 := errorsServer.Receive().Message(message.Get())
	e3 := errorsServer.Send().
		Message(message.Response(http.StatusOK).
			ETag("132r1r312e1").
			Header("someHeader", "someValue"))

	_, e4 := errorsClient.Receive().
		Message(message.Response(http.StatusOK).
			ETag("132r1r312e1").
			Header("someHeader", "wrongValue"))

	checkErrors(t, expectedErrors, e1, e2, e3, e4)
}

// HTTP plain text response payload validation error: payload missing
func TestMissingTextResponsePayloadValidation(t *testing.T) {
	expectedErrors := []string{
		"HTTP client errorsClient: validation error - payload missing - expected [expected payload] but received no payload",
	}

	e1 := errorsClient.Send().
		Message(message.Get().BaseUrl("http://localhost:8083"))

	_, e2 := errorsServer.Receive().Message(message.Get())
	e3 := errorsServer.Send().
		Message(message.Response(http.StatusOK))

	_, e4 := errorsClient.Receive().
		Message(message.Response(http.StatusOK).
			Payload("expected payload"))

	checkErrors(t, expectedErrors, e1, e2, e3, e4)
}

// HTTP plain text response payload validation error: payload invalid
func TestWrongTextResponsePayloadValidation(t *testing.T) {
	expectedErrors := []string{
		"HTTP client errorsClient: validation error - payload mismatch - expected [expected payload] but received [wrong payload]",
	}

	e1 := errorsClient.Send().
		Message(message.Get().BaseUrl("http://localhost:8083"))

	_, e2 := errorsServer.Receive().Message(message.Get())
	e3 := errorsServer.Send().
		Message(message.Response(http.StatusOK).
			Payload("wrong payload"))

	_, e4 := errorsClient.Receive().
		Message(message.Response(http.StatusOK).
			Payload("expected payload"))

	checkErrors(t, expectedErrors, e1, e2, e3, e4)
}
