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
	FactoryContainer
	ControllerContainer() ControllerContainer2
}

func NewRouter() Router {
	return &router{}
}

type router struct {
	http.Handler
	controllers controllerContainer2
	factoryContainer
}

func (r *router) Init() WebError {
	return r.factoryContainer.Init()
}

func (r *router) ControllerContainer() ControllerContainer2 {
	return &r.controllers
}

func (r *router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var (
		begin     = time.Now()
		urlprefix = strings.ToLower(req.URL.Path)
		ctx       = &context{
			request:          req,
			responseWriter:   res,
			factoryContainer: &r.factoryContainer,
		}
		ctl = r.controllers.Get(urlprefix, ctx)
		rts []reflect.Value
		err WebError
	)
	if ctl == nil {
		err = NewWebError(404, "Controller Not found by prefix:%s!", urlprefix)
	} else {
		rts, err = ctl.Call(req.Method)
		if err == nil {
			err = render(rts, ctl)
		} else {
			err.Append(404, "Fail to call `%s`->`%s`", ctl, req.Method)
		}
	}
	if err != nil {
		res.WriteHeader(err.Code())
		res.Write([]byte("<!DOCTYPE html>\r\n<html><head><title>HTTP-Internal Server Error</title></head>"))
		res.Write([]byte("<body><h1>500:HTTP-Internal Server Error</h1><hr/>"))
		res.Write([]byte(fmt.Sprintf("<h3>Error stack:</h3><ul>")))
		for _, _err := range err.Children() {
			res.Write([]byte(fmt.Sprintf("<li>%s</li>", _err.ErrorAll())))
		}
		res.Write([]byte(fmt.Sprintf("</ul><h3>Call stack:</h3><ul>")))
		for _, _nod := range err.CallStack() {
			res.Write([]byte(fmt.Sprintf("<li>%s:%d</li>", _nod.Func, _nod.Line)))
		}
		res.Write([]byte("</ul><hr><h4>Power by GoWeb github.com/ywzjackal/goweb </h4></body></html>"))
	}

	if Debug {
		Log.Printf("%s: %s %d %dus", req.Method, req.URL.Path, 200, time.Now().Sub(begin).Nanoseconds()/1000)
	}
}
