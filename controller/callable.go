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
	_querys     map[string]nodeValue // query parameters
	_standalone []nodeValue          // factory which need be injected after first initialized
	_stateful   []nodeValue          // factory which need be injected from session before called
	_stateless  []nodeValue          // factory which need be injected always new before called
	_actions    map[string]nodeValue // methods wrap
	_init       nodeValue            // Init() function's reflect.Value Pointer
}

func (c *ctlCallable) Call(ctx goweb.Context) goweb.WebError {
	c._interface.SetContext(ctx)
	act, exist := c._actions[ctx.Request().Method]
	if !exist {
		act, exist = c._actions[""]
		if !exist {
			return goweb.NewWebError(http.StatusMethodNotAllowed, "Action `%s` not found! %v", ctx.Request().Method, c._actions)
		}
	}
	ctx_type := ctx.Request().Header.Get("Content-Type")
	if strings.Index(ctx_type, "application/json") >= 0 {
		if err := c.resolveJsonParameters(ctx); err != nil {
			return err.Append(http.StatusBadRequest, "Fail to resolve controller `%s` json parameters!", c._selfValue)
		}
	}
	if err := c.resolveUrlParameters(ctx); err != nil {
		return err.Append(http.StatusInternalServerError, "Fail to resolve controler `%s` url parameters!", c._selfValue)
	}
	if err := resolveInjections(ctx.FactoryContainer(), ctx, c._standalone); err != nil {
		return err.Append(http.StatusInternalServerError, "Fail to resolve standalone injection for %s", c._selfValue)
	}
	if err := resolveInjections(ctx.FactoryContainer(), ctx, c._stateless); err != nil {
		return err.Append(http.StatusInternalServerError, "Fail to resolve stateless injection for %s", c._selfValue)
	}
	if err := resolveInjections(ctx.FactoryContainer(), ctx, c._stateful); err != nil {
		return err.Append(http.StatusInternalServerError, "Fail to resolve stateful injection for %s", c._selfValue)
	}
	act.va.Call(nil)
	return nil
}
