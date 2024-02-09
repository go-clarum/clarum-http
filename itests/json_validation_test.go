package itests

import (
	"github.com/go-clarum/clarum-http/message"
	"net/http"
	"testing"
)

// Client & Server validation
// + @ignore@
func TestJsonOKValidation(t *testing.T) {
	testClient.In(t).Send().
		Message(message.Put().
			Payload("{" +
				"\"active\": true," +
				"\"name\": \"Bruce Wayne\"," +
				"\"age\": 38," +
				"\"height\": 1.879," +
				"\"aliases\": [" +
				"\"Batman\"," +
				"\"The Dark Knight\"" +
				"]," +
				"\"location\": {" +
				"\"street\": \"Mountain Drive\"," +
				"\"number\": 1007," +
				"\"hidden\": true" +
				"}" +
				"}"))

	firstTestServer.In(t).Receive().
		Json().
		Message(message.Put("myApp").
			Payload("{" +
				"\"active\": true," +
				"\"name\": \"Bruce Wayne\"," +
				"\"age\": 38," +
				"\"height\": 1.879," +
				"\"aliases\": [" +
				"\"Batman\"," +
				"\"The Dark Knight\"" +
				"]," +
				"\"location\": {" +
				"\"street\": \"Mountain Drive\"," +
				"\"number\": 1007," +
				"\"hidden\": \"@ignore@\"" +
				"}" +
				"}"))
	firstTestServer.In(t).Send().
		Message(message.Response(http.StatusCreated).
			Payload("{" +
				"\"success\": true," +
				"\"timestamp\": 683546323462" +
				"}"))

	testClient.In(t).Receive().
		Json().
		Message(message.Response(http.StatusCreated).
			Payload("{" +
				"\"success\": true," +
				"\"timestamp\": \"@ignore@\"" +
				"}"))
}
