package message

import (
	"github.com/go-clarum/clarum-http/constants"
	"testing"
)

func TestBuilder(t *testing.T) {
	actual := Response(200).
		ContentType("text/plain").
		ETag("5555").
		Payload("batman!")

	expected := ResponseMessage{
		StatusCode: 200,
		Message: Message{
			MessagePayload: "batman!",
			Headers: map[string]string{
				constants.ContentTypeHeaderName: "text/plain",
				constants.ETagHeaderName:        "5555"},
		},
	}

	if !actual.Equals(&expected) {
		t.Errorf("Message is not as expected.")
	}
}

func TestClone(t *testing.T) {
	message := Response(500).
		ContentType("text/plain").
		ETag("5555").
		Payload("my payload")

	clonedMessage := message.Clone()

	if clonedMessage == message {
		t.Errorf("Message has not been cloned.")
	}

	if !clonedMessage.Equals(message) {
		t.Errorf("Messages are not equal.")
	}
}
