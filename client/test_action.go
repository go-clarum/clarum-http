package client

import (
	"github.com/goclarum/clarum/http/internal"
	"testing"
)

// TestActionBuilder TestSendActionBuilder used to initiate a send or receive action on a client endpoint
// with the context of a test
type TestActionBuilder struct {
	test     *testing.T
	endpoint *Endpoint
}

func (endpoint *Endpoint) In(t *testing.T) *TestActionBuilder {
	return &TestActionBuilder{
		test:     t,
		endpoint: endpoint,
	}
}

func (endpoint *Endpoint) Send() *SendActionBuilder {
	return &SendActionBuilder{
		endpoint: endpoint,
	}
}

func (endpoint *Endpoint) Receive() *ReceiveActionBuilder {
	return &ReceiveActionBuilder{
		endpoint: endpoint,
		options: &receiveOptions{
			expectedPayloadType: internal.Plaintext,
		},
	}
}

func (testBuilder *TestActionBuilder) Send() *TestSendActionBuilder {
	return &TestSendActionBuilder{
		test: testBuilder.test,
		SendActionBuilder: SendActionBuilder{
			endpoint: testBuilder.endpoint,
		},
	}
}

func (testBuilder *TestActionBuilder) Receive() *TestReceiveActionBuilder {
	return &TestReceiveActionBuilder{
		test: testBuilder.test,
		ReceiveActionBuilder: ReceiveActionBuilder{
			endpoint: testBuilder.endpoint,
			options: &receiveOptions{
				expectedPayloadType: internal.Plaintext,
			},
		},
	}
}
