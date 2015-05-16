package goweb

import (
	"reflect"
	"strconv"
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
	// SetContext used by framewrok, no use for user
	SetContext(Context)
	// Type() return one of FactoryTypeStandalone/FactoryTypeStatless/FactoryTypeStatful
	Type() LifeType
	// Call() by request url prefix, if success, []reflect.value contain the method
	// parameters out, else WebError will be set.
	Call(mtd string, ctx Context) ([]reflect.Value, WebError)
}

type controller2 struct {
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

// Context() return the context of client request
func (c *controller2) Context() Context {
	return c._ctx
}

func (c *controller2) SetContext(ctx Context) {
	c._ctx = ctx
}

// Type() return the controller type,one of FactoryTypeStandalone
// FactoryTypeStateless or FactoryTypeStatful
func (c *controller2) Type() LifeType {
	return c._type
}

// InitController when register to controller container before used.
func initController(ctli Controller, fac FactoryContainer) WebError {
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
				fdva.Set(_va.Addr())
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
				return NewWebError(500, "Factory `%s` type is not be specified")
			}
		}
	}
	if ctl.Type() == LifeTypeError {
		return NewWebError(500, "Controller need extend from one of interface ControllerStandalone/ControllerStateful/ControllerStateless")
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

func (c *controller2) Call(mtd string, ctx Context) ([]reflect.Value, WebError) {
	c._ctx = ctx
	act, exist := c._actions[strings.ToLower(mtd)]
	if !exist {
		return nil, NewWebError(404, "Action `%s` not found!", mtd)
	}
	if err := resolveUrlParameters(c, &c._selfValue); err != nil {
		return nil, err.Append(500, "Fail to resolve controler `%s` url parameters!", c._selfValue.Interface())
	}
	if err := resolveInjections(c, c._stateless); err != nil {
		return nil, err.Append(500, "Fail to resolve stateless injection for %s", c._selfValue.Interface())
	}
	if err := resolveInjections(c, c._stateful); err != nil {
		return nil, err.Append(500, "Fail to resolve stateful injection for %s", c._selfValue.Interface())
	}
	rt := act.Call([]reflect.Value{c._selfValue})
	return rt, nil
}

func resolveInjections(c *controller2, nodes []injectNode) WebError {
	var (
		ctx      = c.Context()
		factorys = ctx.FactoryContainer()
	)
	for _, node := range nodes {
		v, err := factorys.Lookup(node.tp, ctx)
		if err != nil {
			return err.Append(500, "Fail to inject `%s` to controller '%s'", node.tp, c)
		}
		node.va.Set(v)
	}
	return nil
}

func resolveUrlParameters(c *controller2, target *reflect.Value) WebError {
	req := c._ctx.Request()
	if err := req.ParseForm(); err != nil {
		return NewWebError(500, "Fail to ParseForm with path:%s,%s", req.URL.String(), err.Error())
	}
	for key, node := range c._querys {
		strs := req.Form[key]
		if len(strs) == 0 {
			continue
		}
		switch node.tp.Kind() {
		case reflect.String:
			node.va.SetString(strs[0])
		case reflect.Bool:
			b, err := strconv.ParseBool(strs[0])
			if err != nil {
				Err.Printf("Fail to convent parameters!\r\nField `%s`(bool) can not set by '%s'", node.tp.Name(), req.Form.Get(key))
				continue
			}
			node.va.SetBool(b)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			num, err := strconv.ParseInt(strs[0], 10, 0)
			if err != nil {
				Err.Printf("Fail to convent parameters!\r\nField `%s`(int) can not set by '%s'", node.tp.Name(), req.Form.Get(key))
				continue
			}
			node.va.SetInt(num)
		case reflect.Float32, reflect.Float64:
			f, err := strconv.ParseFloat(strs[0], 0)
			if err != nil {
				Err.Printf("Fail to convent parameters!\r\nField `%s`(float) can not set by '%s'", node.tp.Name(), req.Form.Get(key))
				continue
			}
			node.va.SetFloat(f)
		case reflect.Slice:
			targetType := node.tp.Elem()
			lens := len(strs)
			values := reflect.MakeSlice(reflect.SliceOf(targetType), lens, lens)
			switch targetType.Kind() {
			case reflect.String:
				values = reflect.ValueOf(strs)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				for j := 0; j < lens; j++ {
					v := values.Index(j)
					intValue, _ := strconv.ParseInt(strs[j], 10, 0)
					v.SetInt(intValue)
				}
			case reflect.Float32, reflect.Float64:
				for j := 0; j < lens; j++ {
					v := values.Index(j)
					floatValue, _ := strconv.ParseFloat(strs[j], 0)
					v.SetFloat(floatValue)
				}
			case reflect.Bool:
				for j := 0; j < lens; j++ {
					v := values.Index(j)
					boolValue, _ := strconv.ParseBool(strs[j])
					v.SetBool(boolValue)
				}
			}
			node.va.Set(values)
		default:
			return NewWebError(500, "Unresolveable url parameter type `%s`", node.tp)
		}
	}
	return nil
}
