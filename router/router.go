package router

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ywzjackal/goweb"
)

const (
	// Never Mind
	__router_name = "goweb.router"
)

// ContextGenerator is a function that create a goweb.Context instance by
// http.ResponseWriter and http.Request.
// This function used by router when new request arrived.
type ContextGenerator func(http.ResponseWriter, *http.Request) goweb.Context

// Buildin Router struct
type router struct {
	http.Handler                           // handle http protocol
	controllers  goweb.ControllerContainer // container of Controllers
	ctxGetor     ContextGenerator          // ContextGenerator used by router
}

// NewRouter is a function return an goweb.Router instance.
func NewRouter(
	c goweb.ControllerContainer,
	ctxGetor ContextGenerator,
) goweb.Router {
	return &router{
		controllers: c,
		ctxGetor:    ctxGetor,
	}
}

func (r *router) Name() string {
	return __router_name
}

func (r *router) ControllerContainer() goweb.ControllerContainer {
	return r.controllers
}

// Router Core
func (r *router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var (
		err goweb.WebError
		_err goweb.WebError
		caller goweb.ControllerCallAble
		begin = time.Now()                    // time of http request arrived.
		urlPrefix = strings.ToLower(req.URL.Path) // url of http request
		ctx = r.ctxGetor(res, req)          // generator a context by http.request and http.response
	)
	// get controller by urlPrefix
	caller, err = r.controllers.Get(urlPrefix, ctx)
	if err != nil || caller == nil {
		// if some thing bad happened, let context know.
		ctx.SetError(err)
		// this is first error when routing request, so try report this error
		// by user defined controller.
		goto ERROR_USER_REPORT
	}
	// Now, controller is ready, try to call action with context.
	// If action not found, err will be return with not nil.
	if err = caller.Call(ctx); err != nil {
		// Action not found! this is second error when routing request,
		// so append this error to top error and let context know.
		err.Append("Fail to call `%s`->`%s`", caller, req.Method)
		ctx.SetError(err)
		goto ERROR_USER_REPORT
	}
	// All things seem to be fine. if some thing bad happened in
	// action, context.error will be defined. so check it.
	// May be error happened in render, so check it.
	if err = ctx.Error(); err != nil {
		// this is first error when routing request, so try report this error
		// by user defined controller.
		goto ERROR_USER_REPORT
	}
	// Well done! Every thing is fine!
	goto FINISH

FINISH:
	// if debug is enabled, print elapsed time of routing this request
	if goweb.Debug {
		goweb.Log.Printf("%s: %s %d %dus", req.Method, req.URL.Path, 200, time.Now().Sub(begin).Nanoseconds()/1000)
	}
	return

ERROR_USER_REPORT:
	// User defined error report.
	// Now, Router can not render a best page to user because some bad thing happened.
	// Router will find a controller with name err.Code() to process this situation.
	// For example:
	// if user defined a controller with name "404", and err.Code() == 404,
	// router will redirect this request to Controller of 404.
caller, _err = r.controllers.Get(fmt.Sprintf("/%d", err.Code()), ctx)
	if _err == nil && caller != nil {
		// Now, we got a User Defined Error Controller fit err.Code(). try to call...
		_err = caller.Call(ctx)
		if _err == nil {
			// If nothing bad happend, render error page.
			res.WriteHeader(err.Code())
			//			_err = render(rts, ctl)
			if _err != nil {
				err.Append(_err.ErrorAll())
			}
		} else {
			// Else render error page by DEFAULT_ERROR_USER_REPORT
			goto DEFAULT_ERROR_USER_REPORT
		}
	} else {
		// If we did not found Controller with Error Code, Try Report Error by
		// DEFAULT_ERROR_USER_REPORT
		goto DEFAULT_ERROR_USER_REPORT
	}
	goto FINISH

DEFAULT_ERROR_USER_REPORT:
	// Now, Try to find a Controller by name "error" and report error.
caller, _err = r.controllers.Get("/error", ctx)
	if _err == nil && caller != nil {
		_err = caller.Call(ctx)
		if _err == nil {
			res.WriteHeader(err.Code())
			//			_err = render(rts, ctl)
			if _err != nil {
				err = _err
				goto DEFAULT_ERROR_REPORT
			}
		} else {
			goto DEFAULT_ERROR_REPORT
		}
	} else {
		goto DEFAULT_ERROR_REPORT
	}
	goto FINISH

DEFAULT_ERROR_REPORT:
	// User not defined ERROR_USER_REPORT Controller or DEFAULT_ERROR_USER_REPORT
	// Controller. so we render an Error Report Basicily
	res.WriteHeader(err.Code())

	res.Write([]byte("<!DOCTYPE html>\r\n<html><head><title>HTTP-Internal Server Error</title></head>"))
	res.Write([]byte(fmt.Sprintf("<body><h1>%d : %s</h1><hr/>", err.Code(), http.StatusText(err.Code()))))

	goweb.Err.Printf("Fail to routing:%s, %d:%s", ctx.Request().URL.Path, err.Code(), http.StatusText(err.Code()))
	res.Write([]byte(fmt.Sprintf("<h3>Error stack:</h3><ul>")))
	goweb.Err.Printf("Error stack:\n %s", err.ErrorAll())
	for _, _err := range err.Children() {
		res.Write([]byte(fmt.Sprintf("<li>%s</li>", _err.Error())))
	}
	res.Write([]byte(fmt.Sprintf("</ul><h3>Call stack:</h3><ul>")))
	for _, _nod := range err.CallStack() {
		res.Write([]byte(fmt.Sprintf("<li>%s:%d</li>", _nod.Func, _nod.Line)))
	}
	res.Write([]byte("</ul><hr><h4>Power by GoWeb github.com/ywzjackal/goweb </h4></body></html>"))
	goto FINISH
}
