package goweb

import (
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

func (r *router) Init() error {
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
		err error
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
		//		res.WriteHeader(505)
		res.Write([]byte(err.Error()))
	}

	if Debug {
		Log.Printf("%s: %s %d %dus", req.Method, req.URL.Path, 200, time.Now().Sub(begin).Nanoseconds()/1000)
	}
}
