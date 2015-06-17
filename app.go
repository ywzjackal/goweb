package goweb

import (
	"fmt"
	"net/http"
	"reflect"
)

type RouterCreator func(scope string, args ...interface{}) Router

var (
	__global_controller_container = make(map[string]ControllerContainer)
	__global_factory_container    = make(map[string]FactoryContainer)
	__global_storages             = make(map[string]Storage)
	__global_router_creators      = make(map[string]RouterCreator)
	__global_routers              = make(map[string]Router)
)

func RegisterController(containerName, path string, ctl Controller) {
	container, alreadyExisted := __global_controller_container[containerName]
	if !alreadyExisted {
		panic(fmt.Sprintf("Controller Container '%s' doesn't exist!", containerName))
	}
	container.Register(path, ctl)
}

func RegisterFactory(containerName string, fac Factory) {
	container, alreadyExisted := __global_factory_container[containerName]
	if !alreadyExisted {
		panic(fmt.Sprintf("Factory Container '%s' doesn't exist!", containerName))
	}
	container.Register(fac)
}

func RegisterRouterCreator(name string, routerCreator RouterCreator) {
	_, alreadyExisted := __global_router_creators[name]
	if alreadyExisted {
		panic("Router(name:" + name + ") duplicate registed!")
	}
	__global_router_creators[name] = routerCreator
	Log.Printf("Register Router Creator `%s`", name)
}

func RegisterRouter(scope string, r Router) {
	_, alreadyExisted := __global_routers[scope]
	if alreadyExisted {
		panic("Router(scope:" + scope + ") duplicate registed!")
	}
	__global_routers[scope] = r
	Log.Printf("Register Router `%s` with scope '%s'", r.Name(), scope)
}

func RouterCreators() map[string]RouterCreator {
	return __global_router_creators
}

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
	New(scope string, args ...interface{}) Router
	Name() string
	Init() WebError
	FactoryContainer() FactoryContainer
	ControllerContainer() ControllerContainer
	MemStorage() Storage
}
