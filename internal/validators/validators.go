package validators

import (
	"errors"
	"fmt"
	"github.com/go-clarum/clarum-core/arrays"
	"github.com/go-clarum/clarum-core/logging"
	clarumstrings "github.com/go-clarum/clarum-core/validators/strings"
	"github.com/go-clarum/clarum-json/comparator"
	"github.com/go-clarum/clarum-json/recorder"
	"github.com/goclarum/clarum/http/internal"
	"github.com/goclarum/clarum/http/message"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
)

func ValidatePath(expectedMessage *message.RequestMessage, actualUrl *url.URL, logger *logging.Logger) error {
	cleanedExpected := cleanPath(expectedMessage.Path)
	cleanedActual := cleanPath(actualUrl.Path)

	if cleanedExpected != cleanedActual {
		return handleError(logger, "validation error - path mismatch - expected [%s] but received [%s]",
			cleanedExpected, cleanedActual)
	} else {
		logger.Info("path validation successful")
	}

	return nil
}

func ValidateHttpMethod(expectedMessage *message.RequestMessage, actualMethod string, logger *logging.Logger) error {
	if expectedMessage.Method != actualMethod {
		return handleError(logger, "validation error - method mismatch - expected [%s] but received [%s]",
			expectedMessage.Method, actualMethod)
	} else {
		logger.Info("method validation successful")
	}

	return nil
}

func ValidateHttpHeaders(expectedMessage *message.Message, actualHeaders http.Header, logger *logging.Logger) error {
	if err := validateHeaders(expectedMessage, actualHeaders); err != nil {
		return handleError(logger, "%s", err)
	} else {
		logger.Info("header validation successful")
	}

	return nil
}

// According to the official specification, HTTP headers must be compared in a case-insensitive way
func validateHeaders(message *message.Message, headers http.Header) error {
	lowerCaseReceivedHeaders := make(map[string][]string)
	for header, values := range headers {
		lowerCaseReceivedHeaders[strings.ToLower(header)] = values
	}

	for header, expectedValue := range message.Headers {
		lowerCaseExpectedHeader := strings.ToLower(header)
		if receivedValues, exists := lowerCaseReceivedHeaders[lowerCaseExpectedHeader]; exists {
			if !arrays.Contains(receivedValues, expectedValue) {
				return errors.New(fmt.Sprintf("validation error - header <%s> mismatch - expected [%s] but received [%s]",
					lowerCaseExpectedHeader, expectedValue, receivedValues))
			}
		} else {
			return errors.New(fmt.Sprintf("validation error - header <%s> missing", lowerCaseExpectedHeader))
		}
	}

	return nil
}

func ValidateHttpQueryParams(expectedMessage *message.RequestMessage, actualUrl *url.URL, logger *logging.Logger) error {
	if err := validateQueryParams(expectedMessage, actualUrl.Query()); err != nil {
		return handleError(logger, "%s", err)
	} else {
		logger.Info("query params validation successful")
	}

	return nil
}

// validate query parameters based on these rules
//
//	-> validate that the param exists
//	-> that the values match
func validateQueryParams(message *message.RequestMessage, params url.Values) error {
	for param, expectedValues := range message.QueryParams {
		if receivedValues, exists := params[param]; exists {
			for _, expectedValue := range expectedValues {
				if !arrays.Contains(receivedValues, expectedValue) {
					return errors.New(fmt.Sprintf("validation error - query param <%s> values mismatch - expected [%v] but received [%s]",
						param, expectedValues, receivedValues))
				}
			}
		} else {
			return errors.New(fmt.Sprintf("validation error - query param <%s> missing", param))
		}
	}

	return nil
}

func ValidateHttpStatusCode(expectedMessage *message.ResponseMessage, actualStatusCode int, logger *logging.Logger) error {
	if actualStatusCode != expectedMessage.StatusCode {
		return handleError(logger, "validation error - status mismatch - expected [%d] but received [%d]",
			expectedMessage.StatusCode, actualStatusCode)
	} else {
		logger.Info("status validation successful")
	}

	return nil
}

func ValidateHttpPayload(expectedMessage *message.Message, actualPayload io.ReadCloser,
	payloadType internal.PayloadType, logger *logging.Logger) error {
	defer closeBody(logger, actualPayload)

	if clarumstrings.IsBlank(expectedMessage.MessagePayload) {
		logger.Info("message payload is empty - no body validation will be done")
		return nil
	}

	bodyBytes, err := io.ReadAll(actualPayload)
	if err != nil {
		return handleError(logger, "could not read response body - %s", err)
	}

	if err := validatePayload(expectedMessage, bodyBytes, payloadType, logger); err != nil {
		return handleError(logger, "%s", err)
	} else {
		logger.Info("payload validation successful")
	}

	return nil
}

func closeBody(logger *logging.Logger, body io.ReadCloser) {
	if err := body.Close(); err != nil {
		logger.Errorf("unable to close body - %s", err)
	}
}

func validatePayload(message *message.Message, actual []byte, payloadType internal.PayloadType, logger *logging.Logger) error {

	if len(actual) == 0 {
		return errors.New(fmt.Sprintf("validation error - payload missing - expected [%s] but received no payload",
			message.MessagePayload))
	} else if payloadType == internal.Plaintext {
		receivedPayload := string(actual)

		if message.MessagePayload != receivedPayload {
			return errors.New(fmt.Sprintf("validation error - payload mismatch - expected [%s] but received [%s]",
				message.MessagePayload, receivedPayload))
		}
	} else if payloadType == internal.Json {
		jsonComparator := comparator.NewComparator().
			Recorder(recorder.NewDefaultRecorder()).
			Build()

		reporterLog, errs := jsonComparator.Compare([]byte(message.MessagePayload), actual)

		if errs != nil {
			logger.Infof("json validation log: %s", reporterLog)
			return errors.New(fmt.Sprintf("json validation errors: [%s]", errs))
		}
		logger.Debugf("json payload validation log: %s", reporterLog)
	}

	return nil
}

func handleError(logger *logging.Logger, format string, a ...any) error {
	errorMessage := fmt.Sprintf(format, a...)
	logger.Errorf(errorMessage)
	return errors.New(errorMessage)
}

// path.Clean() does not remove leading "/", so we do that ourselves
func cleanPath(pathToClean string) string {
	return strings.TrimPrefix(path.Clean(pathToClean), "/")
}
