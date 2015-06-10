package goweb

import (
	"net/http"
	"reflect"
	"strings"
)

type injectNode struct {
	tp reflect.Type   // reflect.type of target
	va *reflect.Value // Pointer reflect.value of target
	id int            // field index
}

type ControllerStandalone interface {
	Controller
}

type ControllerStateless interface {
	Controller
}

type ControllerStateful interface {
	Controller
}

type Controller interface {
	// Context() return current http context
	Context() Context
	// Type() return one of FactoryTypeStandalone/FactoryTypeStatless/FactoryTypeStatful
	Type() LifeType
	// Call() by request url prefix, if success, []reflect.value contain the method
	// parameters out, else WebError will be set.
	Call(mtd string, ctx Context) ([]reflect.Value, WebError)
	// String()
	String() string
}

type controller struct {
	Controller
	_selfValue  reflect.Value
	_ctx        Context               // Realtime Context struct
	_querys     map[string]injectNode // query parameters
	_standalone []injectNode          // factory which need be injected after first initialized
	_stateful   []injectNode          // factory which need be injected from session before called
	_stateless  []injectNode          // factory which need be injected always new before called
	_type       LifeType              // standalone or stateless or stateful
	_actions    map[string]*reflect.Value
}

type controllerExample struct {
	FactoryStandalone
	Controller
	cardid string
}

func (c *controller) String() string {
	t := c._selfValue.Type()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// Context() return the context of client request
func (c *controller) Context() Context {
	return c._ctx
}

// Type() return the controller type,one of FactoryTypeStandalone
// FactoryTypeStateless or FactoryTypeStatful
func (c *controller) Type() LifeType {
	return c._type
}

// InitController when register to controller container before used.
func initController(ctli Controller, fac FactoryContainer) WebError {
	var (
		rtp reflect.Type  = reflect.TypeOf(ctli)
		rva reflect.Value = reflect.ValueOf(ctli)
		ctl *controller   = &controller{
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
				case "ControllerStandalone":
					ctl._type = (LifeTypeStandalone)
				case "ControllerStateful":
					ctl._type = (LifeTypeStateful)
				case "ControllerStateless":
					ctl._type = (LifeTypeStateless)
				default:
				}
				switch {
				case ctlVal.Type().AssignableTo(fdva.Type()):
					fdva.Set(ctlVal)
				default:
					Log.Print(ctlVal.Type(), fdva.Type())
					return NewWebError(500, "interface %s of controller %s can not be assignable!", fdva.Type(), ctlVal.Type())
				}
			}
			continue
		}
	}
	if ctl.Type() == LifeTypeError {
		return NewWebError(500, "Controller need extend from one of interface ControllerStandalone/ControllerStateful/ControllerStateless")
	}
	if err := initSubFields(ctl, rva, fac); err != nil {
		return err
	}
	for i := 0; i < rtp.NumMethod(); i++ {
		mtd := rtp.Method(i)
		if isActionMethod(&mtd) != nil {
			continue
		}
		name := strings.ToLower(mtd.Name[len(ActionPrefix):])
		ctl._actions[name] = &mtd.Func
		Log.Printf("INIT Controller `%s` -> `%s` (%s)", rtp, name, ctli.Type())
	}
	return nil
}

func initSubFields(ctl *controller, rva reflect.Value, fac FactoryContainer) WebError {
	for i := 0; i < rva.NumField(); i++ {
		stfd := rva.Type().Field(i) // struct field
		fdva := rva.Field(i)        // field value
		if !fdva.CanSet() {
			continue
		}
		switch stfd.Type.Kind() {
		case reflect.Int, reflect.String, reflect.Float32, reflect.Bool, reflect.Slice:
			ctl._querys[strings.ToLower(stfd.Name)] = injectNode{
				id: i,
				tp: stfd.Type,
				va: &fdva,
			}
		case reflect.Ptr:
			if isTypeLookupAble(stfd.Type) != nil {
				break
			}
			switch factoryType(stfd.Type) {
			case LifeTypeStandalone:
				ctl._standalone = append(ctl._stateful, injectNode{
					id: i,
					tp: stfd.Type,
					va: &fdva,
				})
				_va, err := fac.Lookup(stfd.Type, nil)
				if err != nil {
					return err.Append(500, "Fail to inject `%s` in `%s`", stfd.Type, ctl._selfValue.Type())
				}
				fdva.Set(_va)
			case LifeTypeStateful:
				ctl._stateful = append(ctl._stateful, injectNode{
					id: i,
					tp: stfd.Type,
					va: &fdva,
				})
			case LifeTypeStateless:
				ctl._stateless = append(ctl._stateful, injectNode{
					id: i,
					tp: stfd.Type,
					va: &fdva,
				})
			default:
				return NewWebError(500, "Factory `%s` type is not be specified", stfd.Type)
			}
		case reflect.Struct:
			initSubFields(ctl, fdva, fac)
		}
	}
	return nil
}

func (c *controller) Call(mtd string, ctx Context) ([]reflect.Value, WebError) {
	c._ctx = ctx
	act, exist := c._actions[strings.ToLower(mtd)]
	if !exist {
		act, exist = c._actions[""]
		if !exist {
			return nil, NewWebError(http.StatusMethodNotAllowed, "Action `%s` not found!", mtd)
		}
	}
	if err := resolveUrlParameters(c, &c._selfValue); err != nil {
		return nil, err.Append(http.StatusInternalServerError, "Fail to resolve controler `%s` url parameters!", c._selfValue)
	}
	if err := resolveInjections(ctx.FactoryContainer(), ctx, c._stateless); err != nil {
		return nil, err.Append(http.StatusInternalServerError, "Fail to resolve stateless injection for %s", c._selfValue)
	}
	if err := resolveInjections(ctx.FactoryContainer(), ctx, c._stateful); err != nil {
		return nil, err.Append(http.StatusInternalServerError, "Fail to resolve stateful injection for %s", c._selfValue)
	}
	rt := act.Call([]reflect.Value{c._selfValue})
	return rt, nil
}
