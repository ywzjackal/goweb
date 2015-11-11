package controller

import (
	"github.com/ywzjackal/goweb"
	"net/http"
	"reflect"
	"strings"
)

type sessionInjectorsContainer struct {
	goweb.Context
	goweb.InjectGetterSetter
}

func (i *sessionInjectorsContainer) Get(name string) goweb.InjectAble {
	a, ok := i.Context.Session().MemMap()[name]
	if !ok {
		return nil
	}
	able, ok := a.(goweb.InjectAble)
	if !ok {
		return nil
	}
	return able
}

func (i *sessionInjectorsContainer) Set(name string, able goweb.InjectAble) {
	i.Context.Session().MemMap()[name] = able
}

// ctl is a implement of goweb.Controller
type ctlCallable struct {
	_interface  goweb.Controller         // Real goweb.Controller
	_selfValue  reflect.Value            //
	_querys     map[string]reflect.Value // query parameters
	_standalone []goweb.InjectNode       // factory which need be injected after first initialized
	_stateful   []goweb.InjectNode       // factory which need be injected from session before called
	_stateless  []goweb.InjectNode       // factory which need be injected always new before called
	_actions    map[string]reflect.Value // methods wrap
	_init       reflect.Value            // Init() function's reflect.Value Pointer
	_preAction  goweb.ActionPreprocessor
	_postAction goweb.ActionPostprocessor
}

func (c *ctlCallable) String() string {
	return c._selfValue.Type().Elem().Name()
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
			return err.Append("Fail to resolve controller `%s` json parameters!", c._selfValue.Type().Elem().Name())
		}
	}
	if err := c.resolveUrlParameters(ctx); err != nil {
		return err.Append("Fail to resolve controler `%s` url parameters!", c._selfValue.Type().Elem().Name())
	}
	if err := injectionStandalone(ctx.FactoryContainer(), c._standalone); err != nil {
		return err.Append("Fail to resolve standalone injection for %s", c._selfValue.Type().Elem().Name())
	}
	if err := injectionStateless(ctx.FactoryContainer(), c._stateless); err != nil {
		return err.Append("Fail to resolve stateless injection for %s", c._selfValue.Type().Elem().Name())
	}
	if err := injectionStateful(ctx.FactoryContainer(), c._stateful, &sessionInjectorsContainer{ctx, nil}); err != nil {
		return err.Append("Fail to resolve stateful injection for %s", c._selfValue.Type().Elem().Name())
	}
	if c._preAction != nil {
		if continue_call := c._preAction.BeforeAction(); !continue_call {
			return c._interface.Context().Error()
		}
	}
	act.Call(nil)
	if c._postAction != nil {
		c._postAction.AfterAction()
	}
	if ctx.Error() != nil {
		return ctx.Error()
	}
	return nil
}
