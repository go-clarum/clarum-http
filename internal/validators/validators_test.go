package validators

import (
	"github.com/go-clarum/clarum-core/config"
	"github.com/go-clarum/clarum-core/logging"
	"github.com/goclarum/clarum/http/constants"
	"github.com/goclarum/clarum/http/message"
	"net/http"
	"testing"
)

var logger = logging.NewLogger(config.LoggingLevel(), "validators test: ")

func TestValidatePathOK(t *testing.T) {
	expectedMessage := createTestMessageWithHeaders()
	req := createRealRequest()

	if err := ValidatePath(expectedMessage, req.URL, logger); err != nil {
		t.Errorf("No header validation error expected, but got %s", err)
	}
}

func TestValidatePathError(t *testing.T) {
	expectedMessage := createTestMessageWithHeaders()
	expectedMessage.Path = "blup/"
	req := createRealRequest()

	err := ValidatePath(expectedMessage, req.URL, logger)

	if err == nil {
		t.Errorf("Path validation error expected, but got none")
	}

	if err.Error() != "validation error - path mismatch - expected [blup] but received [myPath/some/api]" {
		t.Errorf("Path validation error message is unexpected")
	}
}

func TestValidateMethodOK(t *testing.T) {
	expectedMessage := createTestMessageWithHeaders()
	req := createRealRequest()

	if err := ValidateHttpMethod(expectedMessage, req.Method, logger); err != nil {
		t.Errorf("No header validation error expected, but got %s", err)
	}
}

func TestValidateMethodError(t *testing.T) {
	expectedMessage := createTestMessageWithHeaders()
	expectedMessage.Method = http.MethodOptions
	req := createRealRequest()

	err := ValidateHttpMethod(expectedMessage, req.Method, logger)

	if err == nil {
		t.Errorf("Method validation error expected, but got none")
	}

	if err.Error() != "validation error - method mismatch - expected [OPTIONS] but received [POST]" {
		t.Errorf("Path validation error message is unexpected")
	}
}

func TestValidateHeadersOK(t *testing.T) {
	expectedMessage := createTestMessageWithHeaders()
	req := createRealRequest()

	if err := ValidateHttpHeaders(&expectedMessage.Message, req.Header, logger); err != nil {
		t.Errorf("No header validation error expected, but got %s", err)
	}
}

func TestValidateHeaderValueError(t *testing.T) {
	expectedMessage := createTestMessageWithHeaders()
	expectedMessage.Authorization("something else")

	req := createRealRequest()

	err := ValidateHttpHeaders(&expectedMessage.Message, req.Header, logger)

	if err == nil {
		t.Errorf("Header validation error expected, but got none")
	}

	if err.Error() != "validation error - header <authorization> mismatch - expected [something else] but received [[Bearer 0b79bab50daca910b000d4f1a2b675d604257e42]]" {
		t.Errorf("Header validation error message is unexpected")
	}
}

func TestValidateMissingHeaderError(t *testing.T) {
	expectedMessage := createTestMessageWithHeaders()
	expectedMessage.Header("traceid", "124245132")

	req := createRealRequest()

	err := ValidateHttpHeaders(&expectedMessage.Message, req.Header, logger)

	if err == nil {
		t.Errorf("Header validation error expected, but got none")
	}

	if err.Error() != "validation error - header <traceid> missing" {
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

	if err := ValidateHttpQueryParams(expectedMessage, req.URL, logger); err != nil {
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

	err := ValidateHttpQueryParams(expectedMessage, req.URL, logger)
	if err == nil {
		t.Errorf("Query param validation error expected, but got none")
	}

	if err.Error() != "validation error - query param <param2> missing" {
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

	err := ValidateHttpQueryParams(expectedMessage, req.URL, logger)
	if err == nil {
		t.Errorf("Query param validation error expected, but got none")
	}

	if err.Error() != "validation error - query param <param2> values mismatch - expected [[value2]] but received [[value22]]" {
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

	if err := ValidateHttpQueryParams(expectedMessage, req.URL, logger); err != nil {
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

	err := ValidateHttpQueryParams(expectedMessage, req.URL, logger)
	if err == nil {
		t.Errorf("Query param validation error expected, but got none")
	}

	if err.Error() != "validation error - query param <param1> values mismatch - expected [[value1 value2 value4]] but received [[value1 value2 value3]]" {
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
