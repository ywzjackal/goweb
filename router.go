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
	ControllerContainer() ControllerContainer2
	MemStorage() Storage
}

func NewRouter() Router {
	return &router{}
}

type router struct {
	http.Handler
	controllers controllerContainer2
	factorys    factoryContainer
	StorageMemory
}

func (r *router) Init() WebError {
	r.StorageMemory.Init()
	r.ControllerContainer().Init()
	return r.FactoryContainer().Init()
}

func (r *router) ControllerContainer() ControllerContainer2 {
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
		rts []reflect.Value
		err WebError
	)
	session.Init(res, req, r)
	ctl := r.controllers.Get(urlprefix, ctx)
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
		res.Write([]byte(fmt.Sprintf("<body><h1>500:HTTP-Internal Server Error</h1><hr/>")))
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
