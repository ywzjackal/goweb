package goweb

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"
)

var (
	DefaultControllerName = "Default"
	DefaultActionName     = "Default"
)

type Router interface {
	http.Handler
	Init() WebError
	FactoryContainer() FactoryContainer
	ControllerContainer() ControllerContainer
	MemStorage() Storage
}

func NewRouter() Router {
	return &router{}
}

type router struct {
	http.Handler
	controllers controllerContainer
	factorys    factoryContainer
	StorageMemory
}

func (r *router) Init() WebError {
	if err := r.StorageMemory.Init(); err != nil {
		return err
	}
	if err := r.FactoryContainer().Init(); err != nil {
		return err
	}
	if err := r.ControllerContainer().Init(r.FactoryContainer()); err != nil {
		return err
	}
	return nil
}

func (r *router) ControllerContainer() ControllerContainer {
	return &r.controllers
}

func (r *router) FactoryContainer() FactoryContainer {
	return &r.factorys
}

func (r *router) MemStorage() Storage {
	return &r.StorageMemory
}

func (r *router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var (
		begin     = time.Now()
		urlprefix = strings.ToLower(req.URL.Path)
		session   = &session{}
		ctx       = &context{
			request:          req,
			responseWriter:   res,
			factoryContainer: r.FactoryContainer(),
			session:          session,
		}
		rts  []reflect.Value
		err  WebError
		_err WebError
	)

	session.Init(res, req, r)

	ctl, err := r.controllers.Get(urlprefix, ctx)
	if err != nil || ctl == nil {
		ctx.err = err
		goto ERROR_USER_REPORT
	} else {
		rts, err = ctl.Call(req.Method, ctx)
		if err == nil {
			err = render(rts, ctl)
			if err != nil {
				goto ERROR_USER_REPORT
			}
			goto FINISH
		} else {
			err.Append(http.StatusMethodNotAllowed, "Fail to call `%s`->`%s`", ctl, req.Method)
			ctx.err = err
			goto ERROR_USER_REPORT
		}
	}
	goto FINISH

FINISH:
	if Debug {
		Log.Printf("%s: %s %d %dus", req.Method, req.URL.Path, 200, time.Now().Sub(begin).Nanoseconds()/1000)
	}
	return

ERROR_USER_REPORT:
	ctx.request.URL.Path = fmt.Sprintf("/%d", err.Code())
	ctl, _err = r.controllers.Get(ctx.request.URL.Path, ctx)
	if _err == nil && ctl != nil {
		rts, _err = ctl.Call(req.Method, ctx)
		if _err == nil {
			_err = render(rts, ctl)
		} else {
			goto DEFAULT_ERROR_USER_REPORT
		}
	} else {
		goto DEFAULT_ERROR_USER_REPORT
	}
	goto FINISH

DEFAULT_ERROR_USER_REPORT:
	ctx.request.URL.Path = "/error"
	ctl, _err = r.controllers.Get(ctx.request.URL.Path, ctx)
	if _err == nil && ctl != nil {
		rts, _err = ctl.Call(req.Method, ctx)
		if _err == nil {
			_err = render(rts, ctl)
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
