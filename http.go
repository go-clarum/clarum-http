package http

import (
	"github.com/go-clarum/clarum-http/client"
	"github.com/go-clarum/clarum-http/server"
)

type EndpointBuilder struct {
}

func Http() *EndpointBuilder {
	return &EndpointBuilder{}
}

func (heb *EndpointBuilder) Client() *client.EndpointBuilder {
	return client.NewEndpointBuilder()
}

func (heb *EndpointBuilder) Server() *server.EndpointBuilder {
	return server.NewEndpointBuilder()
}
