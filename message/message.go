package message

import (
	"github.com/go-clarum/clarum-http/constants"
	"maps"
)

type Message struct {
	Headers        map[string]string
	MessagePayload string
}

func (message *Message) header(key string, value string) *Message {
	if message.Headers == nil {
		message.Headers = make(map[string]string)
	}

	message.Headers[key] = value
	return message
}

func (message *Message) contentType(value string) *Message {
	return message.header(constants.ContentTypeHeaderName, value)
}

func (message *Message) authorization(value string) *Message {
	return message.header(constants.AuthorizationHeaderName, value)
}

func (message *Message) eTag(value string) *Message {
	return message.header(constants.ETagHeaderName, value)
}

func (message *Message) payload(payload string) *Message {
	message.MessagePayload = payload
	return message
}

func (message *Message) clone() Message {
	return Message{
		Headers:        maps.Clone(message.Headers),
		MessagePayload: message.MessagePayload,
	}
}
