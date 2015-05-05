package goweb

import (
	"net/http"
)

type Context interface {
	Request() *http.Request
	ResponseWriter() http.ResponseWriter
	FactoryContainer() FactoryContainer
	ControllerName() string
	ActionName() string
	SetTitle(string)
}

type context struct {
	FactoryStateless
	Context
	request          *http.Request
	responseWriter   http.ResponseWriter
	factoryContainer FactoryContainer
	controllerName   string
	actionName       string
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

func (c *context) ControllerName() string {
	return c.controllerName
}

func (c *context) ActionName() string {
	return c.actionName
}
