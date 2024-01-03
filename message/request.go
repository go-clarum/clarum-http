package message

import (
	"fmt"
	"github.com/goclarum/clarum/http/internal/utils"
	"maps"
	"net/http"
	"reflect"
)

type RequestMessage struct {
	Message
	Method      string
	Url         string
	Path        string
	QueryParams map[string][]string
}

func Get(pathElements ...string) *RequestMessage {
	return &RequestMessage{
		Method: http.MethodGet,
		Path:   utils.BuildPath("", pathElements...),
	}
}

func Head(pathElements ...string) *RequestMessage {
	return &RequestMessage{
		Method: http.MethodHead,
		Path:   utils.BuildPath("", pathElements...),
	}
}

func Post(pathElements ...string) *RequestMessage {
	return &RequestMessage{
		Method: http.MethodPost,
		Path:   utils.BuildPath("", pathElements...),
	}
}

func Put(pathElements ...string) *RequestMessage {
	return &RequestMessage{
		Method: http.MethodPut,
		Path:   utils.BuildPath("", pathElements...),
	}
}

func Delete(pathElements ...string) *RequestMessage {
	return &RequestMessage{
		Method: http.MethodDelete,
		Path:   utils.BuildPath("", pathElements...),
	}
}

func Options(pathElements ...string) *RequestMessage {
	return &RequestMessage{
		Method: http.MethodOptions,
		Path:   utils.BuildPath("", pathElements...),
	}
}

func Trace(pathElements ...string) *RequestMessage {
	return &RequestMessage{
		Method: http.MethodTrace,
		Path:   utils.BuildPath("", pathElements...),
	}
}

func Patch(pathElements ...string) *RequestMessage {
	return &RequestMessage{
		Method: http.MethodPatch,
		Path:   utils.BuildPath("", pathElements...),
	}
}

// BaseUrl - While this should normally be configured only on the HTTP client,
// this is also allowed on the message so that a client can send a request to different targets.
// When used on a message passed to an HTTP server, it will do nothing.
func (request *RequestMessage) BaseUrl(baseUrl string) *RequestMessage {
	request.Url = baseUrl
	return request
}

func (request *RequestMessage) Header(key string, value string) *RequestMessage {
	request.Message.header(key, value)
	return request
}

func (request *RequestMessage) ContentType(value string) *RequestMessage {
	request.Message.contentType(value)
	return request
}

func (request *RequestMessage) Authorization(value string) *RequestMessage {
	request.Message.authorization(value)
	return request
}

func (request *RequestMessage) QueryParam(key string, values ...string) *RequestMessage {
	if request.QueryParams == nil {
		request.QueryParams = make(map[string][]string)
	}

	if _, exists := request.QueryParams[key]; exists {
		for _, value := range values {
			request.QueryParams[key] = append(request.QueryParams[key], value)
		}
	} else {
		request.QueryParams[key] = values
	}

	return request
}

func (request *RequestMessage) Payload(payload string) *RequestMessage {
	request.Message.MessagePayload = payload
	return request
}

func (request *RequestMessage) Clone() *RequestMessage {
	return &RequestMessage{
		Method:      request.Method,
		Url:         request.Url,
		Path:        request.Path,
		QueryParams: maps.Clone(request.QueryParams),
		Message:     request.Message.clone(),
	}
}

func (request *RequestMessage) Equals(other *RequestMessage) bool {

	if request.Method != other.Method {
		return false
	} else if request.Url != other.Url {
		return false
	} else if request.Path != other.Path {
		return false
	} else if !maps.Equal(request.Headers, other.Headers) {
		return false
	} else if !reflect.DeepEqual(request.QueryParams, other.QueryParams) {
		return false
	} else if request.MessagePayload != other.MessagePayload {
		return false
	}
	return true
}

func (request *RequestMessage) ToString() string {
	return fmt.Sprintf(
		"["+
			"Method: %s, "+
			"BaseUrl: %s, "+
			"Path: '%s', "+
			"Headers: %s, "+
			"QueryParams: %s, "+
			"MessagePayload: %s"+
			"]",
		request.Method, request.Url, request.Path,
		request.Headers, request.QueryParams, request.MessagePayload)
}
