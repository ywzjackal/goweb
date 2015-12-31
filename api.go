package goweb

import (
	"net/http"
	"reflect"
	"time"
)

// Session interface for browser
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

// Context interface for http request to response
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

// ControllerContainer is a container of controllers
type ControllerContainer interface {
	// Register a new controller caller to container with its prefix.
	// prefix is url prefix
	Register(prefix string, caller Controller)
	// Get controller by url prefix
	// return nil if not found in container
	Get(prefix string, ctx Context) (ControllerCallAble, WebError)
}

type Controller interface {
	// Type() return one of FactoryTypeStandalone/FactoryTypeStateless/FactoryTypeStateful
	Type() LifeType
	// SetContext() set current context before invoke actions methods
	SetContext(Context)
	// Context() return current http context
	Context() Context
}

type ActionPreprocessor interface {
	// BeforeAction() filter all request before action method
	// return false will stop framework to continue call action method
	// return true for normal
	BeforeAction() bool
}

type ActionPostprocessor interface {
	// AfterAction() will be called when action method finished.
	AfterAction()
}

type ControllerSchema interface {
	// NewCallAble return a struct implement `ControllerCallAble`, used by router be invoked.
	NewCallAble() ControllerCallAble
	// Type() return one of FactoryTypeStandalone/FactoryTypeStateless/FactoryTypeStateful
	Type() LifeType
}

type ControllerCallAble interface {
	// Call() Controller Action by Context
	Call(ctx Context) WebError
}

type InitAble interface {
	Init()
}

type DestroyAble interface {
	Destroy()
}

type InjectNode struct {
	Name  string        // full struct name with package path
	Value reflect.Value //
}

type InjectAble interface {
	FullName() string               //
	Alias() string                  //
	Type() LifeType                 //
	ReflectType() reflect.Type      //
	ReflectValue() reflect.Value    //
	FieldsStandalone() []InjectNode //
	FieldsStateful() []InjectNode   //
	FieldsStateless() []InjectNode  //
	Target() Factory
}

type Factory interface {
	Type() LifeType
}

type InjectGetterSetter interface {
	Get(string) InjectAble
	Set(string, InjectAble)
}

// FactoryContainer is the container of factories
type FactoryContainer interface {
	// Init after create new instance.
	// notice:
	// 		fc := &FactoryContainerStruct{}
	// 		fc.Init() //！！Must be called after created！！
	// 初始化，当工厂容器被创建后，必须马上使用本函数初始化
	Init() WebError
	// Register an injectable factory with its alias to container， set alias to "" for ignore.
	// 注册工厂并以指定的名字命名, 如果想使用默认名菜，将名称的参数设置为“”即可
	Register(Factory, string)
	//
	LookupStandalone(string) (InjectAble, WebError)
	//
	LookupStateless(string) (InjectAble, WebError)
	//
	LookupStateful(string, InjectGetterSetter) (InjectAble, WebError)
	//
	Lookup(string, InjectGetterSetter) (Factory, WebError)
	//
	LookupType(string) LifeType
}

// RouterInterface
type Router interface {
	http.Handler
	// ControllerContainer() return interface of controller container used by this router
	ControllerContainer() ControllerContainer
}

// Storage interface
type Storage interface {
	// Get() return the element(interface{}) find by key,
	// return nil if not found with the key
	Get(string) interface{}
	// Set() element(interface{}) with it's key,
	// and data will removed after the default duration from last query
	Set(string, interface{})
	// SetWithLife() set data with deadline.
	SetWithLife(string, interface{}, time.Duration)
	// Remove() element before deadline.
	Remove(string)
}
