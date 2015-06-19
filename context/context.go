package context

import (
	"net/http"

	"github.com/ywzjackal/goweb"
)

type context struct {
	goweb.Context
	request          *http.Request
	responseWriter   http.ResponseWriter
	factoryContainer goweb.FactoryContainer
	session          goweb.Session
	err              goweb.WebError
}

func NewContext(res http.ResponseWriter, req *http.Request, fac goweb.FactoryContainer, sess goweb.Session) goweb.Context {
	return &context{
		request:          req,
		responseWriter:   res,
		factoryContainer: fac,
		session:          sess,
	}
}

func (c *context) Request() *http.Request {
	return c.request
}

func (c *context) ResponseWriter() http.ResponseWriter {
	return c.responseWriter
}

func (c *context) FactoryContainer() goweb.FactoryContainer {
	return c.factoryContainer
}

func (c *context) Session() goweb.Session {
	return c.session
}

func (c *context) Error() goweb.WebError {
	if c.err == nil {
		c.err = goweb.NewWebError(0, "SUCCESS")
	}
	return c.err
}

func (c *context) SetError(err goweb.WebError) {
	c.err = err
}
