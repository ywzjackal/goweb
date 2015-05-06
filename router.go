package goweb

import (
	"fmt"
	"net/http"
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
	ControllerContainer
}

func NewRouter() Router {
	return &router{}
}

type router struct {
	http.Handler
	controllerContainer
	factoryContainer
	controllers map[string]Controller
}

func (r *router) Init() WebError {
	return r.factoryContainer.Init()
}

func (r *router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	//	defer func() {
	//		if r := recover(); r != nil {
	//			res.Write([]byte(fmt.Sprintf("%s", r)))
	//		}
	//	}()
	var (
		begin    = time.Now()
		path     = req.URL.Path
		array    = strings.Split(path, "/")
		arrayLen = len(array)
		context  = &context{
			request:          req,
			responseWriter:   res,
			factoryContainer: &r.factoryContainer,
		}
		err WebError
	)
	if arrayLen > 1 {
		context.controllerName = strings.ToLower(array[1])
		if arrayLen > 2 {
			context.actionName = array[2]
		}
	}
	if strings.TrimSpace(context.controllerName) == "" {
		context.controllerName = DefaultControllerName
	}
	if strings.TrimSpace(context.actionName) == "" {
		context.actionName = DefaultActionName
	}
	err = r.Call(context)
	if err != nil {
		res.WriteHeader(err.Code())
		res.Write([]byte("<!DOCTYPE html>\r\n<html><head><title>HTTP-Internal Server Error</title></head>"))
		res.Write([]byte("<body><h1>500:HTTP-Internal Server Error</h1><hr/>"))
		res.Write([]byte(fmt.Sprintf("<h3>Error stack:</h3><ul>")))
		for _, _err := range err.Children() {
			res.Write([]byte(fmt.Sprintf("<li style:'padding-left:100px'>%s</li>", _err.ErrorAll())))
		}
		res.Write([]byte(fmt.Sprintf("</ul><h3>Call stack:</h3><ul>")))
		for _, _nod := range err.CallStack() {
			res.Write([]byte(fmt.Sprintf("<li style:'padding-left:100px'>%s:%d</li>", _nod.Func, _nod.Line)))
		}
		res.Write([]byte("</ul><hr><h4>Power by GoWeb github.com/ywzjackal/goweb </h4></body></html>"))
	}

	if Debug {
		Log.Printf("%s: %s %d %dus", req.Method, req.URL.Path, 200, time.Now().Sub(begin).Nanoseconds()/1000)
	}
}
