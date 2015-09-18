package controller

import (
	"github.com/ywzjackal/goweb"
	"net/http"
	"reflect"
	"strings"
)

type nodeValue struct {
	va reflect.Value // Pointer reflect.value of target
	id int           // field index
}

// ctl is a implement of goweb.Controller
type ctlCallable struct {
	_interface  goweb.Controller     // Real goweb.Controller
	_parent     nodeValue            // All custom controller's parent is controllerValue
	_selfValue  reflect.Value        //
	_selfType   reflect.Type         //
	_ctx        goweb.Context        // Realtime goweb.Context struct
	_querys     map[string]nodeValue // query parameters
	_standalone []nodeValue          // factory which need be injected after first initialized
	_stateful   []nodeValue          // factory which need be injected from session before called
	_stateless  []nodeValue          // factory which need be injected always new before called
	_actions    map[string]nodeValue // methods wrap
	_init       nodeValue            // Init() function's reflect.Value Pointer
}

func (c *ctlCallable) Init() {

}

func (c *ctlCallable) String() string {
	t := c._selfValue.Type()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

func (c *ctlCallable) Context() goweb.Context {
	return c._ctx
}

func (c *ctlCallable) Call(ctx goweb.Context) ([]reflect.Value, goweb.WebError) {
	c._ctx = ctx
	act, exist := c._actions[strings.ToLower(ctx.Request().Method)]
	if !exist {
		act, exist = c._actions[""]
		if !exist {
			return nil, goweb.NewWebError(http.StatusMethodNotAllowed, "Action `%s` not found!", ctx.Request().Method)
		}
	}
	ctx_type := ctx.Request().Header.Get("Content-Type")
	if strings.Index(ctx_type, "application/json") >= 0 {
		if err := c.resolveJsonParameters(); err != nil {
			return nil, err.Append(http.StatusBadRequest, "Fail to resolve controller `%s` json parameters!", c._selfValue)
		}
	}
	if err := c.resolveUrlParameters(); err != nil {
		return nil, err.Append(http.StatusInternalServerError, "Fail to resolve controler `%s` url parameters!", c._selfValue)
	}
	if err := resolveInjections(ctx.FactoryContainer(), ctx, c._stateless); err != nil {
		return nil, err.Append(http.StatusInternalServerError, "Fail to resolve stateless injection for %s", c._selfValue)
	}
	if err := resolveInjections(ctx.FactoryContainer(), ctx, c._stateful); err != nil {
		return nil, err.Append(http.StatusInternalServerError, "Fail to resolve stateful injection for %s", c._selfValue)
	}
	rt := act.va.Call([]reflect.Value{c._selfValue})
	return rt, nil
}
