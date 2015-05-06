package goweb

import (
	"net/http"
	"reflect"
)

type Controller interface {
	http.Handler
}

type actionWrap struct {
	index          int
	actionName     string
	name           string
	method         *reflect.Method
	context        reflect.Value
	parameters     []reflect.Value
	parameterTypes []reflect.Type
}

type controller struct {
	Controller
}

func (c *controller) ServeHTTP(res http.ResponseWriter, req *http.Request) {

}

func render(rets []reflect.Value, c Context) WebError {
	if len(rets) == 0 {
		return NewWebError(1, "Controller Action need return a ViewType like `html`,`json`.")
	}
	viewType, ok := rets[0].Interface().(string)
	if !ok {
		return NewWebError(1, "Controller Action need return a ViewType of string! but got `%s`", rets[0].Type())
	}
	view, isexist := views[viewType]
	if !isexist {
		return NewWebError(1, "Unknow ViewType :%s", viewType)
	}
	interfaces := make([]interface{}, len(rets)-1, len(rets)-1)
	for i, ret := range rets[1:] {
		interfaces[i] = ret.Interface()
	}
	return view.Render(c, interfaces...)
}
