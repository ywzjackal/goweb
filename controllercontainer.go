package goweb

import "reflect"

// ControllerContainer is a container to store controllers
type ControllerContainer2 interface {
	Init() WebError
	// Register a new controller to container
	// prefix is url prefix
	Register(prefix string, ctl Controller2)
	// Get controller by url prefix
	// return nil if not found in container
	Get(prefix string, ctx Context) Controller2
}

// controllerContainer is buildin default controller container
type controllerContainer2 struct {
	ControllerContainer2
	ctls map[string]Controller2
}

func (c *controllerContainer2) Init() WebError {
	c.ctls = make(map[string]Controller2)
	return nil
}

func (c *controllerContainer2) Register(prefix string, ctl Controller2) {
	_, exist := c.ctls[prefix]
	if exist {
		panic("URL Prefix:" + prefix + " register duplicated")
	}
	err := InitController(ctl)
	if err != nil {
		panic(err.ErrorAll())
	}
	c.ctls[prefix] = ctl
}

func (c *controllerContainer2) Get(prefix string, ctx Context) Controller2 {
	ctl, exist := c.ctls[prefix]
	if !exist || ctl == nil {
		return nil
	}
	switch ctl.Type() {
	case FactoryTypeStandalone:
		ctl.SetContext(ctx)
		return ctl
	case FactoryTypeStateless:
		ctl = reflect.New(reflect.TypeOf(ctl).Elem()).Interface().(Controller2)
		if err := InitController(ctl); err != nil {
			panic(err)
		} else {
			ctl.SetContext(ctx)
			return ctl
		}
	case FactoryTypeStateful:
		mem := ctx.Session().MemMap()
		itfs, isexist := mem["__ctl_"+prefix]
		if !isexist {
			ctl = reflect.New(reflect.TypeOf(ctl).Elem()).Interface().(Controller2)
			if err := InitController(ctl); err != nil {
				panic(err)
			}
			mem["__ctl_"+prefix] = ctl
			ctl.SetContext(ctx)
			return ctl
		}
		ctl, ok := itfs.(Controller2)
		if !ok {
			panic("kljlkjljl;jl;jl;jljlj")
		}
		ctl.SetContext(ctx)
		return ctl
	}
	return nil
}
