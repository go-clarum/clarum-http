package client

import (
	"github.com/go-clarum/clarum-http/internal"
	"github.com/go-clarum/clarum-http/message"
	"net/http"
	"testing"
)

type receiveOptions struct {
	expectedPayloadType internal.PayloadType
}

// ReceiveActionBuilder used to configure a receive action on a client endpoint without the context of a test
// the method chain will end with the .Message() method which will return an error.
// The error will be a problem encountered during receiving or a validation error.
type ReceiveActionBuilder struct {
	endpoint *Endpoint
	options  *receiveOptions
}

// TestReceiveActionBuilder used to configure a receive action on a client endpoint with the context of a test
// the method chain will end with the .Message() method which will not return anything.
// Any error encountered during receiving or validating will fail the test by calling t.Error().
type TestReceiveActionBuilder struct {
	test *testing.T
	ReceiveActionBuilder
}

func (testBuilder *TestReceiveActionBuilder) Json() *TestReceiveActionBuilder {
	testBuilder.options.expectedPayloadType = internal.Json
	return testBuilder
}

func (builder *ReceiveActionBuilder) Json() *ReceiveActionBuilder {
	builder.options.expectedPayloadType = internal.Json
	return builder
}

func (testBuilder *TestReceiveActionBuilder) Message(message *message.ResponseMessage) {
	if _, err := testBuilder.endpoint.receive(message, *testBuilder.options); err != nil {
		testBuilder.test.Error(err)
	}
}

func (builder *ReceiveActionBuilder) Message(message *message.ResponseMessage) (*http.Response, error) {
	return builder.endpoint.receive(message, *builder.options)
}
