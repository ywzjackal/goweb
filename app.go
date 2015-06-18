package goweb

import (
	"net/http"
	"reflect"
)

type Context interface {
	Request() *http.Request
	ResponseWriter() http.ResponseWriter
	FactoryContainer() FactoryContainer
	SetTitle(string)
	Session() Session
	Error() WebError
}

type Controller interface {
	// Context() return current http context
	Context() Context
	// Type() return one of FactoryTypeStandalone/FactoryTypeStatless/FactoryTypeStatful
	Type() LifeType
	// Call() by request url prefix, if success, []reflect.value contain the method
	// parameters out, else WebError will be set.
	Call(mtd string, ctx Context) ([]reflect.Value, WebError)
	// String()
	String() string
}

type Router interface {
	http.Handler
	Name() string
	FactoryContainer() FactoryContainer
	ControllerContainer() ControllerContainer
	MemStorage() Storage
}
