package goweb

import (
	"fmt"
	"reflect"
	"strings"
)

type Controller interface {
}

type actionWrap struct {
	index          int
	actionName     string
	name           string
	method         *reflect.Method
	parameters     []reflect.Value
	parameterTypes []reflect.Type
}

type controllerWrap struct {
	controllerName string
	name           string
	fullName       string
	actions        map[string]*actionWrap
	controller     *controller
}

type controller struct {
	Controller
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

func (c *controllerContainer) Call(context Context) WebError {
	var (
		err error
	)
	cname := strings.ToLower(ControllerPrefix + context.ControllerName())
	aname := strings.ToLower(ActionPrefix + context.ActionName())
	cwp, ok := c.controllers[cname]
	if !ok {
		return NewWebError(404, "Controller Not Found:"+cname, nil)
	}
	awp, ok := cwp.actions[aname]
	if !ok {
		return NewWebError(404, cname+" doesn't have "+aname, nil)
	}
	switch len(awp.parameters) {
	case 1:
	case 2:
		if err := context.Request().ParseForm(); err != nil {
			return NewWebError(505,
				"Fail to convent http.request to ParametersInterface!\r\n", NewWebError(1, err.Error(), nil))
		}
		awp.parameters[1], err = paramtersFromRequestUrl(awp.parameterTypes[1], context)
		if err != nil {
			return NewWebError(505, err.Error(), nil)
		}
	default:
		if err := context.Request().ParseForm(); err != nil {
			return NewWebError(505, "Fail to convent http.request to ParametersInterface!", NewWebError(1, err.Error(), nil))
		}
		awp.parameters[1], err = paramtersFromRequestUrl(awp.parameterTypes[1], context)
		if err != nil {
			return NewWebError(505, err.Error(), nil)
		}
		// Enjection
		for i, t := range awp.parameterTypes[2:] {
			factory, err := context.FactoryContainer().Lookup(t, context)
			if err != nil {
				return NewWebError(505, fmt.Sprintf("Lookup Fail:`%s`\r\n\t%s", t, err.Error()), nil)
			} else {
				Log.Printf("Lookup Success:`%s`", t)
			}
			awp.parameters[i+2] = factory
		}
	}
	returnValues := awp.method.Func.Call(awp.parameters)
	err = render(returnValues, context)
	if err != nil {
		return NewWebError(505, err.Error(), nil)
	}
	return nil
}

func render(rets []reflect.Value, c Context) error {
	if len(rets) == 0 {
		return fmt.Errorf("Controller Action need return a ViewType like `html`,`json`.")
	}
	viewType, ok := rets[0].Interface().(string)
	if !ok {
		return fmt.Errorf("Controller Action need return a ViewType of string! but got `%s`", rets[0].Type())
	}
	view, isexist := views[viewType]
	if !isexist {
		return fmt.Errorf("Unknow ViewType :%s", viewType)
	}
	interfaces := make([]interface{}, len(rets)-1, len(rets)-1)
	for i, ret := range rets[1:] {
		interfaces[i] = ret.Interface()
	}
	return view.Render(c, interfaces...)
}
