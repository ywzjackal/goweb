package goweb

import (
	"net/http"
)

type Context interface {
	Request() *http.Request
	ResponseWriter() http.ResponseWriter
	FactoryContainer() FactoryContainer
	SetTitle(string)
	Session() Session
	Error() WebError
}

type context struct {
	FactoryStateless
	Context
	request          *http.Request
	responseWriter   http.ResponseWriter
	factoryContainer FactoryContainer
	session          Session
	err              WebError
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

func (c *context) Session() Session {
	return c.session
}

func (c *context) Error() WebError {
	if c.err == nil {
		c.err = NewWebError(0, "SUCCESS")
	}
	return c.err
}
