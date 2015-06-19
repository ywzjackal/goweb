package controller

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/ywzjackal/goweb"
)

const (
	ActionPrefix = "Action"
)

type injectNode struct {
	tp reflect.Type   // reflect.type of target
	va *reflect.Value // Pointer reflect.value of target
	id int            // field index
}

type ControllerStandalone interface {
	goweb.Controller
}

type ControllerStateless interface {
	goweb.Controller
}

type ControllerStateful interface {
	goweb.Controller
}

type controller struct {
	goweb.Controller
	_selfValue  reflect.Value
	_ctx        goweb.Context         // Realtime goweb.Context struct
	_querys     map[string]injectNode // query parameters
	_standalone []injectNode          // factory which need be injected after first initialized
	_stateful   []injectNode          // factory which need be injected from session before called
	_stateless  []injectNode          // factory which need be injected always new before called
	_type       goweb.LifeType        // standalone or stateless or stateful
	_actions    map[string]*reflect.Value
}

func (c *controller) String() string {
	t := c._selfValue.Type()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// Context() return the context of client request
func (c *controller) Context() goweb.Context {
	return c._ctx
}

// Type() return the controller type,one of FactoryTypeStandalone
// FactoryTypeStateless or FactoryTypeStatful
func (c *controller) Type() goweb.LifeType {
	return c._type
}

func (c *controller) Call(mtd string, ctx goweb.Context) ([]reflect.Value, goweb.WebError) {
	c._ctx = ctx
	act, exist := c._actions[strings.ToLower(mtd)]
	if !exist {
		act, exist = c._actions[""]
		if !exist {
			return nil, goweb.NewWebError(http.StatusMethodNotAllowed, "Action `%s` not found!", mtd)
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

// InitController when register to controller container before used.
func initController(ctli goweb.Controller, fac goweb.FactoryContainer) {
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
					ctl._type = (goweb.LifeTypeStandalone)
				case "ControllerStateful":
					ctl._type = (goweb.LifeTypeStateful)
				case "ControllerStateless":
					ctl._type = (goweb.LifeTypeStateless)
				default:
					continue
				}
				switch {
				case ctlVal.Type().AssignableTo(fdva.Type()):
					fdva.Set(ctlVal)
				default:
					panic(fmt.Sprintf("interface %s can not be assignable by %s !", fdva.Type(), ctlVal.Type()))
				}
			}
			continue
		}
	}
	if ctl.Type() == goweb.LifeTypeError {
		panic("goweb.Controller need extend from one of interface ControllerStandalone/ControllerStateful/ControllerStateless")
	}
	if err := initSubFields(ctl, rva, fac); err != nil {
		panic(err.ErrorAll())
	}
	for i := 0; i < rtp.NumMethod(); i++ {
		mtd := rtp.Method(i)
		if isActionMethod(&mtd) != nil {
			continue
		}
		name := strings.ToLower(mtd.Name[len(ActionPrefix):])
		ctl._actions[name] = &mtd.Func
		goweb.Log.Printf("INIT goweb.Controller `%s` -> `%s` (%s)", rtp, name, ctli.Type())
	}
}

func initSubFields(ctl *controller, rva reflect.Value, fac goweb.FactoryContainer) goweb.WebError {
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
			case goweb.LifeTypeStandalone:
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
			case goweb.LifeTypeStateful:
				ctl._stateful = append(ctl._stateful, injectNode{
					id: i,
					tp: stfd.Type,
					va: &fdva,
				})
			case goweb.LifeTypeStateless:
				ctl._stateless = append(ctl._stateful, injectNode{
					id: i,
					tp: stfd.Type,
					va: &fdva,
				})
			default:
				return goweb.NewWebError(500, "Factory `%s` type is not be specified", stfd.Type)
			}
		case reflect.Struct:
			initSubFields(ctl, fdva, fac)
		}
	}
	return nil
}

func isActionMethod(method *reflect.Method) goweb.WebError {
	if !strings.HasPrefix(method.Name, ActionPrefix) {
		return goweb.NewWebError(500, "func %s doesn't have prefix '%s'", method.Name, ActionPrefix)
	}
	if method.Type.NumIn() != 1 {
		err := goweb.NewWebError(500, "Action func %s need function without parameters in! got %d", method.Name, method.Type.NumIn()-1)
		goweb.Err.Print(err.Error())
		return err
	}
	if method.Type.NumOut() == 0 || method.Type.Out(0).Kind() != reflect.String {
		err := goweb.NewWebError(500, "Action func %s need function with at least on parameters out, the first parameters out must be string to specify which view do you need,like 'json','txt','html' etc.", method.Name)
		goweb.Err.Print(err.Error())
		return err
	}
	return nil
}

func isTypeLookupAble(rt reflect.Type) goweb.WebError {
	if rt.Kind() == reflect.Interface {
		return nil
	}
	if rt.Kind() == reflect.Ptr && rt.Elem().Kind() == reflect.Struct {
		return nil
	}
	return goweb.NewWebError(500, "`%s(%s)` is not a interface or *struct", rt, reflectType(rt))
}

var (
	factoryTypesByName = map[string]goweb.LifeType{
		"FactoryStandalone": goweb.LifeTypeStandalone,
		"FactoryStateful":   goweb.LifeTypeStateful,
		"FactoryStateless":  goweb.LifeTypeStateless,
	}
)

func factoryType(t reflect.Type) goweb.LifeType {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return goweb.LifeTypeError
	}
	for name, ft := range factoryTypesByName {
		if _, b := t.FieldByName(name); b {
			return ft
		}
	}
	return goweb.LifeTypeError
}

func resolveInjections(factorys goweb.FactoryContainer, ctx goweb.Context, nodes []injectNode) goweb.WebError {
	for _, node := range nodes {
		v, err := factorys.Lookup(node.tp, ctx)
		if err != nil {
			return err.Append(500, "Fail to inject `%s`", node.tp)
		}
		if !v.IsValid() {
			return goweb.NewWebError(500, "inject invalid value to `%s`", node.va)
		}
		if v.Kind() != reflect.Ptr {
			return goweb.NewWebError(500, "inject invalid type of %s, need Ptr", v.Kind())
		}
		node.va.Set(v)
	}
	return nil
}

func reflectType(rt reflect.Type) string {
	typstr := ""
	for rt.Kind() == reflect.Ptr {
		typstr = typstr + "*"
		rt = rt.Elem()
	}
	return typstr + rt.Kind().String()
}
