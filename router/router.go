package router

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/ywzjackal/goweb"
	"github.com/ywzjackal/goweb/view"
)

const (
	// Never Mind
	__router_name = "goweb.router"
)

var (
	// DefaultControllerName define the Controller when Router can not find
	// a Controller fit the http request will be used.
	DefaultControllerName = "Default"
	// DefaultActionName define the Action when Router can not find a Action
	// fit the http request.
	DefaultActionName = "Default"
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
		// time of http request arrived.
		begin = time.Now()
		// url of http request
		urlprefix = strings.ToLower(req.URL.Path)
		// generator a context by http.request and http.response
		ctx = r.ctxGetor(res, req)
		// store
//		rts []reflect.Value
		// never mind
		err goweb.WebError
		// never mind
		_err goweb.WebError
	)
	// get controller by urlprefix
	ctl, err := r.controllers.Get(urlprefix, ctx)
	if err != nil || ctl == nil {
		// if some thing bad happend, let context know.
		ctx.SetError(err)
		// this is first error when routing request, so try report this error
		// by user defined controller.
		goto ERROR_USER_REPORT
	} else {
		// Now, controller is ready, try to call action whith context.
		// If action not found, err will be return with not nil.
//		rts, err = ctl.Call(ctx)
		if err == nil {
			// All things seem to be fine. if some thing bad happend in
			// action, context.error will be defined. so check it.
			if ctx.Error() == nil {
				// finialize,  render page to user.
//				err = render(rts, ctl)
			} else {
				// or get the error.
				err = ctx.Error()
			}
			// May be error happened in render, so check it.
			if err == nil {
				// Well done! Every thing is fine!
				goto FINISH
			} else {
				// this is first error when routing request, so try report this error
				// by user defined controller.
				goto ERROR_USER_REPORT
			}
		} else {
			// Action not found! this is second error when routing request,
			// so append this error to top error and let context know.
			err.Append(http.StatusMethodNotAllowed, "Fail to call `%s`->`%s`", ctl, req.Method)
			ctx.SetError(err)
			goto ERROR_USER_REPORT
		}
	}

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
	ctx.Request().URL.Path = fmt.Sprintf("/%d", err.Code())
	ctl, _err = r.controllers.Get(ctx.Request().URL.Path, ctx)
	if _err == nil && ctl != nil {
		// Now, we got a User Defined Error Controller fit err.Code(). try to call...
//		rts, _err = ctl.Call(ctx)
		if _err == nil {
			// If nothing bad happend, render error page.
			res.WriteHeader(err.Code())
//			_err = render(rts, ctl)
			if _err != nil {
				err.Append(_err.Code(), _err.ErrorAll())
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
	ctx.Request().URL.Path = "/error"
	ctl, _err = r.controllers.Get(ctx.Request().URL.Path, ctx)
	if _err == nil && ctl != nil {
//		rts, _err = ctl.Call(ctx)
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
	res.Write([]byte(fmt.Sprintf("<h3>Error stack:</h3><ul>")))
	for _, _err := range err.Children() {
		res.Write([]byte(fmt.Sprintf("<li>%s</li>", _err.ErrorAll())))
	}
	res.Write([]byte(fmt.Sprintf("</ul><h3>Call stack:</h3><ul>")))
	for _, _nod := range err.CallStack() {
		res.Write([]byte(fmt.Sprintf("<li>%s:%d</li>", _nod.Func, _nod.Line)))
	}
	res.Write([]byte("</ul><hr><h4>Power by GoWeb github.com/ywzjackal/goweb </h4></body></html>"))
	goto FINISH
}

func render(rets []reflect.Value, c goweb.Controller) goweb.WebError {
	if len(rets) == 0 {
		return goweb.NewWebError(1, "Controller Action need return a ViewType like `html`,`json`.")
	}
	viewType, ok := rets[0].Interface().(string)
	if !ok {
		return goweb.NewWebError(1, "Controller Action need return a ViewType of string! but got `%s`", rets[0].Type())
	}
	view := view.GetView(viewType)
	if view == nil {
		return goweb.NewWebError(1, "Unknow ViewType :%s", viewType)
	}
	interfaces := make([]interface{}, len(rets)-1, len(rets)-1)
	for i, ret := range rets[1:] {
		interfaces[i] = ret.Interface()
	}
	err := view.Render(c, interfaces...)
	if err != nil {
		return err.Append(500, "Fail to render view %s, data:%+v", viewType, interfaces)
	}
	return nil
}
