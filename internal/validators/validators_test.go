package validators

import (
	"github.com/goclarum/clarum/http/constants"
	"github.com/goclarum/clarum/http/message"
	"net/http"
	"testing"
)

func TestValidatePathOK(t *testing.T) {
	expectedMessage := createTestMessageWithHeaders()
	req := createRealRequest()

	if err := ValidatePath("pathOKTest", expectedMessage, req.URL); err != nil {
		t.Errorf("No header validation error expected, but got %s", err)
	}
}

func TestValidatePathError(t *testing.T) {
	expectedMessage := createTestMessageWithHeaders()
	expectedMessage.Path = "blup/"
	req := createRealRequest()

	err := ValidatePath("pathErrorTest", expectedMessage, req.URL)

	if err == nil {
		t.Errorf("Path validation error expected, but got none")
	}

	if err.Error() != "pathErrorTest: validation error - HTTP path mismatch - expected [blup] but received [myPath/some/api]" {
		t.Errorf("Path validation error message is unexpected")
	}
}

func TestValidateMethodOK(t *testing.T) {
	expectedMessage := createTestMessageWithHeaders()
	req := createRealRequest()

	if err := ValidateHttpMethod("methodOKTest", expectedMessage, req.Method); err != nil {
		t.Errorf("No header validation error expected, but got %s", err)
	}
}

func TestValidateMethodError(t *testing.T) {
	expectedMessage := createTestMessageWithHeaders()
	expectedMessage.Method = http.MethodOptions
	req := createRealRequest()

	err := ValidateHttpMethod("methodErrorTest", expectedMessage, req.Method)

	if err == nil {
		t.Errorf("Method validation error expected, but got none")
	}

	if err.Error() != "methodErrorTest: validation error - HTTP method mismatch - expected [OPTIONS] but received [POST]" {
		t.Errorf("Path validation error message is unexpected")
	}
}

func TestValidateHeadersOK(t *testing.T) {
	expectedMessage := createTestMessageWithHeaders()
	req := createRealRequest()

	if err := ValidateHttpHeaders("headersOKTest", &expectedMessage.Message, req.Header); err != nil {
		t.Errorf("No header validation error expected, but got %s", err)
	}
}

func TestValidateHeaderValueError(t *testing.T) {
	expectedMessage := createTestMessageWithHeaders()
	expectedMessage.Authorization("something else")

	req := createRealRequest()

	err := ValidateHttpHeaders("headersErrorTest", &expectedMessage.Message, req.Header)

	if err == nil {
		t.Errorf("Header validation error expected, but got none")
	}

	if err.Error() != "headersErrorTest: validation error - header <authorization> mismatch - expected [something else] but received [[Bearer 0b79bab50daca910b000d4f1a2b675d604257e42]]" {
		t.Errorf("Header validation error message is unexpected")
	}
}

func TestValidateMissingHeaderError(t *testing.T) {
	expectedMessage := createTestMessageWithHeaders()
	expectedMessage.Header("traceid", "124245132")

	req := createRealRequest()

	err := ValidateHttpHeaders("headersErrorTest", &expectedMessage.Message, req.Header)

	if err == nil {
		t.Errorf("Header validation error expected, but got none")
	}

	if err.Error() != "headersErrorTest: validation error - header <traceid> missing" {
		t.Errorf("Header validation error message is unexpected")
	}
}

func TestValidateQueryParamsOK(t *testing.T) {
	expectedMessage := message.Get("myPath").
		QueryParam("param1", "value1").
		QueryParam("param2", "value2")

	req := createRealRequest()
	qParams := req.URL.Query()
	qParams.Set("param1", "value1")
	qParams.Set("param2", "value2")
	req.URL.RawQuery = qParams.Encode()

	if err := ValidateHttpQueryParams("queryParamsOKTest", expectedMessage, req.URL); err != nil {
		t.Errorf("No query param validation error expected, but got %s", err)
	}
}

func TestValidateQueryParamsParamMismatch(t *testing.T) {
	expectedMessage := message.Get("myPath").
		QueryParam("param1", "value1").
		QueryParam("param2", "value2")

	req := createRealRequest()
	qParams := req.URL.Query()
	qParams.Set("param1", "value1")
	qParams.Set("param3", "value2")
	req.URL.RawQuery = qParams.Encode()

	err := ValidateHttpQueryParams("queryParamsErrorTest", expectedMessage, req.URL)
	if err == nil {
		t.Errorf("Query param validation error expected, but got none")
	}

	if err.Error() != "queryParamsErrorTest: validation error - query param <param2> missing" {
		t.Errorf("Query param validation error message is unexpected")
	}
}

func TestValidateQueryParamsValueMismatch(t *testing.T) {
	expectedMessage := message.Get("myPath").
		QueryParam("param1", "value1").
		QueryParam("param2", "value2")

	req := createRealRequest()
	qParams := req.URL.Query()
	qParams.Set("param1", "value1")
	qParams.Set("param2", "value22")
	req.URL.RawQuery = qParams.Encode()

	err := ValidateHttpQueryParams("queryParamValueErrorTest", expectedMessage, req.URL)
	if err == nil {
		t.Errorf("Query param validation error expected, but got none")
	}

	if err.Error() != "queryParamValueErrorTest: validation error - query param <param2> values mismatch - expected [[value2]] but received [[value22]]" {
		t.Errorf("Query param validation error message is unexpected")
	}
}

func TestValidateQueryParamsMultiValueOK(t *testing.T) {
	expectedMessage := message.Get("myPath").
		QueryParam("param1", "value1").
		QueryParam("param1", "value3")

	req := createRealRequest()
	qParams := req.URL.Query()
	qParams.Set("param1", "value1")
	qParams.Add("param1", "value2")
	qParams.Add("param1", "value3")
	req.URL.RawQuery = qParams.Encode()

	if err := ValidateHttpQueryParams("queryParamsMultiValueOKTest", expectedMessage, req.URL); err != nil {
		t.Errorf("No query param validation error expected, but got %s", err)
	}
}

func TestValidateQueryParamsMultiValueMismatch(t *testing.T) {
	expectedMessage := message.Get("myPath").
		QueryParam("param1", "value1", "value2", "value4")

	req := createRealRequest()
	qParams := req.URL.Query()
	qParams.Set("param1", "value1")
	qParams.Add("param1", "value2")
	qParams.Add("param1", "value3")
	req.URL.RawQuery = qParams.Encode()

	err := ValidateHttpQueryParams("queryParamsMultiValueErrorTest", expectedMessage, req.URL)
	if err == nil {
		t.Errorf("Query param validation error expected, but got none")
	}

	if err.Error() != "queryParamsMultiValueErrorTest: validation error - query param <param1> values mismatch - expected [[value1 value2 value4]] but received [[value1 value2 value3]]" {
		t.Errorf("Query param validation error message is unexpected")
	}
}

func createTestMessageWithHeaders() *message.RequestMessage {
	return message.Post("myPath", "", "some", "/", "api").
		Header("Connection", "keep-alive").
		ContentType("application/json").
		Authorization("Bearer 0b79bab50daca910b000d4f1a2b675d604257e42")
}

func createRealRequest() *http.Request {
	req, _ := http.NewRequest(http.MethodPost, "myPath/some/api", nil)
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set(constants.ContentTypeHeaderName, "application/json")
	req.Header.Set(constants.AuthorizationHeaderName, "Bearer 0b79bab50daca910b000d4f1a2b675d604257e42")

	return req
}
