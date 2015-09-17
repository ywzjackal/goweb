package goweb

import (
	"net/http"
	"reflect"
	"time"
)

// Session
type Session interface {
	// Id() return session's id
	Id() string
	// Get(key) session value by key
	Get(string) string
	// Set(key,value) to session
	Set(string, string)
	// Remove(key) by key
	Remove(string)
	// MemMap() return low-level memory map
	MemMap() map[interface{}]interface{}
}

// Context
type Context interface {
	// Request() return pointer of http.Request of http request
	Request() *http.Request
	// ResponseWriter() return interface http.ResponseWriter of http response
	ResponseWriter() http.ResponseWriter
	// FactoryContainer() return interface of factory container
	FactoryContainer() FactoryContainer
	// Session() return user session if session has been set
	Session() Session
	// Error() WebError of Context, Can be reset by SetError() method.
	Error() WebError
	// SetError(err) of Context, Can be get by Error() method.
	SetError(WebError)
}

// ControllerContainer is a container to store controllers
type ControllerContainer interface {
	// Register a new controller to container
	// prefix is url prefix
	Register(prefix string, ctl Controller)
	// Get controller by url prefix
	// return nil if not found in container
	Get(prefix string, ctx Context) (Controller, WebError)
}

type Controller interface {
	// Init() Will be called :
	//
	//   if Controller is Standalone, called when Framework initialization;
	//   if Controller is Stateless,  called when Request initialization;
	//   if Controller is Statefull,  called at first use in session;
	Init()
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

type Factory interface{}

// FactoryContainer is the container interface of factorys
type FactoryContainer interface {
	// Init after create new instanse.
	// notice:
	// 		fc := &FactoryContainerStruct{}
	// 		fc.Init() //！！Must be called after created！！
	// 初始化，当工厂容器被创建后，必须马上使用本函数初始化
	Init() WebError
	// RegisterFactory will register a new factory for Lookup later.
	// 注册工厂，以供以后查询（Lookup）使用
	Register(Factory) WebError
	// Lookup factory by type from this container or depends and return it.
	// Lookup also enject Ptr or Interface fields which is Exported and
	// Setable for the factory be looking up.
	// if something wrong happend, the error will be set.
	// for Standalone Factory, Lookup will return the global factory which
	// can be assignable to the type of rt.
	// for Stateful Factory, Lookup will find in session, if it doesn't exist
	// in session, Lookup will create new one and add to session for later.
	// for Stateless Factory, Lookup will allways create new instance by rt,
	// and never store in container or session.
	// 根据工厂的类型（reflect.type）从容器或depends中查找与之相配的值并返回。
	// 如果工厂中有导出并且可以被赋值的指针（Prt）或接口（Interface）域（Field），
	// Lookup 自动注册这些域。
	// 对于独立工厂，Lookup将从全局唯一的实例返回，
	// 对于有状态工厂，Lookup将先从会话中查找，如果没有将创建一个新的工厂实例并添加到会话
	// 中，以供以后的使用。
	// 对于无状态工厂，Lookup始终返回一个刚被新创建的实例。
	Lookup(rt reflect.Type, context Context) (reflect.Value, WebError)
}

// RouterInterface
type Router interface {
	http.Handler
	// return interface of controller container
	ControllerContainer() ControllerContainer
}

// View is the top of view component's interface, all custom view component need
// implament from this, and realize method Render(Controller, ...interface{}) WebError
//
// View 是视图的顶级接口组件，所有的自定义视图组件必须实现此接口
type View interface {
	Render(Controller, ...interface{}) WebError
}

// Storage interface
type Storage interface {
	// Get() return the element(interface{}) find by key,
	// return nil if not found with the key
	Get(string) interface{}
	// Set() element(interface{}) with it's key,
	// and data will removed after the default duration from last query
	Set(string, interface{})
	// Set() element(interface{}) with life.
	// data will be removed after the duration from last query
	SetWithLife(string, interface{}, time.Duration)
	// Remove() element before deadline.
	Remove(string)
}
