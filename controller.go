package goweb

import (
	"errors"
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
	RegisterController(string, Controller)
	Controller(string) Controller
	Call(Context) error
}

type controllerContainer struct {
	ControllerContainer
	controllers map[string]*controllerWrap
}

func (c *controllerContainer) RegisterController(name string, ci Controller) {
	if c.controllers == nil {
		c.controllers = make(map[string]*controllerWrap)
	}
	cw := initControllerWrap(name, ci)
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

func (c *controllerContainer) Call(context Context) error {
	cname := strings.ToLower(ControllerPrefix + context.ControllerName())
	cwp, ok := c.controllers[cname]
	if !ok {
		return errors.New("Controller Not Found:" + cname)
	}
	aname := strings.ToLower(ActionPrefix + context.ActionName())
	awp, ok := cwp.actions[aname]
	if !ok {
		return errors.New(cname + " doesn't have " + aname)
	}
	switch len(awp.parameters) {
	case 1:
	case 2:
		if err := context.Request().ParseForm(); err != nil {
			panic("Fail to convent http.request to ParametersInterface!\r\n" + err.Error())
		}
		awp.parameters[1] = paramtersFromRequestUrl(awp.parameterTypes[1], context.Request().Form)
	default:
		if err := context.Request().ParseForm(); err != nil {
			panic("Fail to convent http.request to ParametersInterface!\r\n" + err.Error())
		}
		awp.parameters[1] = paramtersFromRequestUrl(awp.parameterTypes[1], context.Request().Form)
		// Enjection
		for i, t := range awp.parameterTypes[2:] {
			factory, err := context.FactoryContainer().Lookup(t, context)
			if err != nil {
				panic(fmt.Sprintf("Lookup Fail:`%s`\r\n\t%s", t, err.Error()))
			} else {
				Log.Printf("Lookup Success:`%s`", t)
			}
			awp.parameters[i+2] = factory
		}
	}
	returnValues := awp.method.Func.Call(awp.parameters)
	err := render(returnValues, context)
	return err
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
