package goweb

import (
	"reflect"
	"strings"
)

type injectNode struct {
	tp reflect.Type   // reflect.type of target
	va *reflect.Value // Pointer reflect.value of target
	id int            // field index
}

type Controller2 interface {
	// Context() return current http context
	Context() Context
	// Type() return one of FactoryTypeStandalone/FactoryTypeStatless/FactoryTypeStatful
	Type() FactoryType
	// Call() by request url prefix, if success, []reflect.value contain the method
	// parameters out, else WebError will be set.
	Call(mtd string) ([]reflect.Value, WebError)
}

type controller2 struct {
	_selfValue  reflect.Value
	_ctx        Context               // Realtime Context struct
	_querys     map[string]injectNode // query parameters
	_standalone []injectNode          // factory which need be injected from global
	_stateful   []injectNode          // factory which need be injected from session
	_stateless  []injectNode          // factory which need be injected always new
	_type       FactoryType           // standalone or stateless or stateful
	_actions    map[string]*reflect.Value
}

type controllerExample struct {
	FactoryStandalone
	Controller2
	cardid string
}

// Context() return the context of client request
func (c *controller2) Context() Context {
	return c._ctx
}

// Type() return the controller type,one of FactoryTypeStandalone
// FactoryTypeStateless or FactoryTypeStatful
func (c *controller2) Type() FactoryType {
	return c._type
}

// InitController when register to controller container before used.
func InitController(ctli Controller2) WebError {
	var (
		rtp reflect.Type  = reflect.TypeOf(ctli)
		rva reflect.Value = reflect.ValueOf(ctli)
		ctl *controller2  = &controller2{
			_selfValue: rva,
			_querys:    make(map[string]injectNode),
			_actions:   make(map[string]*reflect.Value),
		}
		ctlVal reflect.Value = reflect.ValueOf(ctl)
	)
	for rva.Kind() == reflect.Ptr {
		rva = rva.Elem()
	}
	for i := 0; i < rva.NumField(); i++ {
		stfd := rva.Type().Field(i) // struct field
		fdva := rva.Field(i)        // field value
		if !fdva.CanSet() {
			continue
		}
		if stfd.Anonymous {
			if fdva.Type().Kind() == reflect.Interface {
				switch fdva.Type().Name() {
				case "FactoryStandalone":
					ctl._type = (FactoryTypeStandalone)
				case "FactoryStateful":
					ctl._type = (FactoryTypeStateful)
				case "FactoryStateless":
					ctl._type = (FactoryTypeStateless)
				default:
					if ctlVal.Type().AssignableTo(fdva.Type()) {
						fdva.Set(ctlVal)
					} else {
						Log.Print(ctlVal.Type(), fdva.Type())
					}
				}
			}
			continue
		}
		switch stfd.Type.Kind() {
		case reflect.Int, reflect.String, reflect.Float32, reflect.Bool, reflect.Slice:
			ctl._querys[strings.ToLower(stfd.Name)] = injectNode{
				id: i,
				tp: stfd.Type,
				va: &fdva,
			}
		case reflect.Interface, reflect.Ptr:
			if isTypeLookupAble(stfd.Type) != nil {
				break
			}
			switch factoryType(stfd.Type) {
			case FactoryTypeStandalone:
				ctl._standalone = append(ctl._stateful, injectNode{
					id: i,
					tp: stfd.Type,
					va: &fdva,
				})
			case FactoryTypeStateful:
				ctl._stateful = append(ctl._stateful, injectNode{
					id: i,
					tp: stfd.Type,
					va: &fdva,
				})
			case FactoryTypeStateless:
				ctl._stateless = append(ctl._stateful, injectNode{
					id: i,
					tp: stfd.Type,
					va: &fdva,
				})
			default:
				return NewWebError(500, "Factory `%s` type is not be specified")
			}
		}
	}
	if ctl.Type() == FactoryTypeError {
		return NewWebError(500, "Controller need extend from one of interface FactoryStandalone/FactoryStateful/FactoryStateless")
	}
	for i := 0; i < rtp.NumMethod(); i++ {
		mtd := rtp.Method(i)
		if isActionMethod(&mtd) != nil {
			continue
		}
		name := strings.ToLower(mtd.Name[len(ActionPrefix):])
		ctl._actions[name] = &mtd.Func
		Log.Printf("Register Controller `%s` -> `%s` (%s)", rtp, name, ctli.Type())
	}
	return nil
}

func (c *controller2) Call(mtd string) ([]reflect.Value, WebError) {
	mtd = strings.ToLower(mtd)
	act, exist := c._actions[mtd]
	var self reflect.Value
	switch c._type {
	case FactoryTypeStandalone:
		self = c._selfValue
	case FactoryTypeStateless:
		self = reflect.New(c._selfValue.Type().Elem())
	case FactoryTypeStateful:
		// inject from session
	}
	if !exist {
		return nil, NewWebError(404, "Action `%s` not found!", mtd)
	}
	rt := act.Call([]reflect.Value{self})
	return rt, nil
}
