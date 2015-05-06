package goweb

import (
	"reflect"
	"strings"
)

type controllerWrap struct {
	controllerName string
	name           string
	fullName       string
	actions        map[string]*actionWrap
	controller     *controller
}

type ControllerContainer interface {
	RegisterController(Controller)
	Controller(string) Controller
	Call(Context) WebError
}

type controllerContainer struct {
	ControllerContainer
	controllers map[string]*controllerWrap
}

func (c *controllerContainer) RegisterController(ci Controller) {
	if c.controllers == nil {
		c.controllers = make(map[string]*controllerWrap)
	}
	cw := initControllerWrap(ci)
	Log.Println("Register Controller `" + cw.name + "`")
	c.controllers[cw.name] = cw
}

func (c *controllerContainer) Controller(name string) Controller {
	if c.controllers == nil {
		c.controllers = make(map[string]*controllerWrap)
	}
	cwp, ok := c.controllers[name]
	if ok {
		return cwp.controller
	}
	return nil
}

func (c *controllerContainer) Call(ctx Context) WebError {
	var (
		err        WebError
		callParams []reflect.Value
	)
	cname := strings.ToLower(ControllerPrefix + ctx.ControllerName())
	aname := strings.ToLower(ActionPrefix + ctx.ActionName())
	cwp, ok := c.controllers[cname]
	if !ok {
		return NewWebError(404, "Controller Not Found:"+cname)
	}
	awp, ok := cwp.actions[aname]
	if !ok {
		return NewWebError(404, cname+" doesn't have "+aname)
	}
	callParams, err = lookupAndInject(awp.parameterTypes, ctx)
	if err != nil {
		return err.Append(500, "Fail to Call %s:%s", cname, aname)
	}
	returnValues := awp.method.Func.Call(callParams)
	err = render(returnValues, ctx)
	if err != nil {
		return NewWebError(505, err.Error(), nil)
	}
	return nil
}
