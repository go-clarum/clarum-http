package client

import (
	"time"
)

type EndpointBuilder struct {
	baseUrl     string
	contentType string
	name        string
	timeout     time.Duration
}

func NewEndpointBuilder() *EndpointBuilder {
	return &EndpointBuilder{}
}

func (builder *EndpointBuilder) Name(name string) *EndpointBuilder {
	builder.name = name
	return builder
}

func (builder *EndpointBuilder) BaseUrl(baseUrl string) *EndpointBuilder {
	builder.baseUrl = baseUrl
	return builder
}

func (builder *EndpointBuilder) ContentType(contentType string) *EndpointBuilder {
	builder.contentType = contentType
	return builder
}

func (builder *EndpointBuilder) Timeout(timeout time.Duration) *EndpointBuilder {
	builder.timeout = timeout
	return builder
}

func (builder *EndpointBuilder) Build() *Endpoint {
	return newEndpoint(builder.name, builder.baseUrl, builder.contentType, builder.timeout)
}
