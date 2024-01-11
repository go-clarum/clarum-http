package server

import (
	"time"
)

type EndpointBuilder struct {
	contentType string
	port        uint
	name        string
	timeout     time.Duration
}

func NewEndpointBuilder() *EndpointBuilder {
	return &EndpointBuilder{}
}

func (builder *EndpointBuilder) Timeout(timeout time.Duration) *EndpointBuilder {
	builder.timeout = timeout
	return builder
}

func (builder *EndpointBuilder) Name(name string) *EndpointBuilder {
	builder.name = name
	return builder
}

func (builder *EndpointBuilder) Port(port uint) *EndpointBuilder {
	builder.port = port
	return builder
}

func (builder *EndpointBuilder) ContentType(contentType string) *EndpointBuilder {
	builder.contentType = contentType
	return builder
}

func (builder *EndpointBuilder) Build() *Endpoint {
	return newServerEndpoint(builder.name, builder.port, builder.contentType, builder.timeout)
}
