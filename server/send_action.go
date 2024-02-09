package server

import (
	"github.com/go-clarum/clarum-http/message"
	"testing"
)

// SendActionBuilder used to configure a send action on a server endpoint without the context of a test
// the method chain will end with the .Message() method which will return an error.
// The error will be a problem encountered during sending.
type SendActionBuilder struct {
	endpoint *Endpoint
}

// TestSendActionBuilder used to configure a send action on a server endpoint with the context of a test
// the method chain will end with the .Message() method which will not return anything.
// Any error encountered during sending will fail the test by calling t.Error().
type TestSendActionBuilder struct {
	test *testing.T
	SendActionBuilder
}

func (testBuilder *TestSendActionBuilder) Message(message *message.ResponseMessage) {
	if err := testBuilder.endpoint.send(message); err != nil {
		testBuilder.test.Error(err)
	}
}

func (builder *SendActionBuilder) Message(message *message.ResponseMessage) error {
	return builder.endpoint.send(message)
}
