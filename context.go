package goweb

import (
	"net/http"
)

type Context interface {
	Request() *http.Request
	ResponseWriter() http.ResponseWriter
	FactoryContainer() FactoryContainer
	SetTitle(string)
}

type context struct {
	FactoryStateless
	Context
	request          *http.Request
	responseWriter   http.ResponseWriter
	factoryContainer FactoryContainer
}

func (c *context) Request() *http.Request {
	return c.request
}

func (c *context) ResponseWriter() http.ResponseWriter {
	return c.responseWriter
}

func (c *context) FactoryContainer() FactoryContainer {
	return c.factoryContainer
}
