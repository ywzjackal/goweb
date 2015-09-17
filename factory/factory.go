package factory

import (
	"reflect"

	"github.com/ywzjackal/goweb"
)

const (
	FactoryAutoInitFuncName = "init_auto"
)

var (
	factoryTypesByName = map[string]goweb.LifeType{
		"FactoryStandalone": goweb.LifeTypeStandalone,
		"FactoryStateful":   goweb.LifeTypeStateful,
		"FactoryStateless":  goweb.LifeTypeStateless,
	}
)

// FactoryStandalone is the factory that only one instance in system，all the
// Enjection share on the same factory instance. so be carful for thread-safe
//
// 独立状态工厂接口，仅有一个实例存在与系统中，所有注入共享一个实例，所以必要时应该考虑线程
// 安全。
type FactoryStandalone interface {
	goweb.Factory
}

// FactoryStateful is stateful for user in session. each session has the one
// and only instance for itself, no shared, and will auto destroy when
// session timeout or be destroyed.
//
// 有状态工厂接口，面向用户的有状态工厂，每个会话（SESSION）包含一个与众不同的实例，不共
// 享，当会话（SESSION）超时或摧毁时包含的有状态的工厂也将被自动摧毁
type FactoryStateful interface {
	goweb.Factory
}

// FactoryStateless is stateless for user in session. Enjection will allways
// create a new factory instance for using.
//
// 无状态工厂接口，面向用户无状态，每次注入（Enject）将创建一个新的实例以供调用。
type FactoryStateless interface {
	goweb.Factory
}

func factoryType(t reflect.Type) goweb.LifeType {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return goweb.LifeTypeError
	}
	for name, ft := range factoryTypesByName {
		if _, b := t.FieldByName(name); b {
			return ft
		}
	}
	return goweb.LifeTypeError
}

type injectNode struct {
	tp reflect.Type   // reflect.type of target
	va *reflect.Value // Pointer reflect.value of target
	id int            // field index
}

type factory struct {
	goweb.Factory
	_selfValue reflect.Value
	_intFunc   reflect.Value
	//	_querys     map[string]injectNode // query parameters
	_standalone []injectNode   // factory which need be injected after first initialized
	_stateful   []injectNode   // factory which need be injected from session before called
	_stateless  []injectNode   // factory which need be injected always new before called
	_type       goweb.LifeType // standalone or stateless or stateful
	//	_actions    map[string]*reflect.Value
}

func (f *factory) Init() {
	if f._intFunc.IsValid() {
		f._intFunc.Call([]reflect.Value{})
	} else {
		goweb.Log.Printf("`%s` doesn't have method `Init`!", f._selfValue.Type())
	}
}

func isTypeRegisterAble(rt reflect.Type) goweb.WebError {
	if rt.Kind() == reflect.Ptr && rt.Elem().Kind() == reflect.Struct {
		return nil
	}
	return goweb.NewWebError(500, "`%s(%s)` is not a *struct", rt, reflectType(rt))
}

func isTypeLookupAble(rt reflect.Type) goweb.WebError {
	if rt.Kind() == reflect.Interface {
		return nil
	}
	if rt.Kind() == reflect.Ptr && rt.Elem().Kind() == reflect.Struct {
		return nil
	}
	return goweb.NewWebError(500, "`%s(%s)` is not a interface or *struct", rt, reflectType(rt))
}

func reflectType(rt reflect.Type) string {
	typstr := ""
	for rt.Kind() == reflect.Ptr {
		typstr = typstr + "*"
		rt = rt.Elem()
	}
	return typstr + rt.Kind().String()
}
