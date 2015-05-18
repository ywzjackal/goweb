package goweb

import "reflect"

// ControllerContainer is a container to store controllers
type ControllerContainer interface {
	Init(FactoryContainer) WebError
	// Register a new controller to container
	// prefix is url prefix
	Register(prefix string, ctl Controller)
	// Get controller by url prefix
	// return nil if not found in container
	Get(prefix string, ctx Context) (Controller, WebError)
}

// controllerContainer is buildin default controller container
type controllerContainer struct {
	ControllerContainer
	ctls     map[string]Controller
	factorys FactoryContainer
}

func (c *controllerContainer) Init(factorys FactoryContainer) WebError {
	c.ctls = make(map[string]Controller)
	c.factorys = factorys
	return nil
}

func (c *controllerContainer) Register(prefix string, ctl Controller) {
	_, exist := c.ctls[prefix]
	if exist {
		panic("URL Prefix:" + prefix + " register duplicated")
	}
	err := initController(ctl, c.factorys)
	if err != nil {
		panic(err.ErrorAll())
	}
	c.ctls[prefix] = ctl
}

func (c *controllerContainer) Get(prefix string, ctx Context) (Controller, WebError) {
	var (
		ctl Controller = nil
		ok  bool       = false
	)
	ctl, ok = c.ctls[prefix]
	if !ok || ctl == nil {
		return nil, NewWebError(404, "Controller not found!")
	}
	switch ctl.Type() {
	case LifeTypeStandalone:
	case LifeTypeStateless:
		ctl = reflect.New(reflect.TypeOf(ctl).Elem()).Interface().(Controller)
		if err := initController(ctl, ctx.FactoryContainer()); err != nil {
			return nil, err.Append(500, "Fail to auto initialize stateless controller!")
		}
	case LifeTypeStateful:
		mem := ctx.Session().MemMap()
		itfs, ok := mem["__ctl_"+prefix]
		if !ok {
			ctl = reflect.New(reflect.TypeOf(ctl).Elem()).Interface().(Controller)
			if err := initController(ctl, ctx.FactoryContainer()); err != nil {
				return nil, err.Append(500, "Fail to auto initialize stateful controller!")
			}
			mem["__ctl_"+prefix] = ctl
		} else {
			ctl, ok = itfs.(Controller)
			if !ok {
				return nil, NewWebError(500, "Fail to convert session stateful interface to controller!")
			}
		}
	default:
		return nil, NewWebError(500, "Controller Life Status Undefined!")
	}
	return ctl, nil
}
