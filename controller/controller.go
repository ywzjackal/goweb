package controller

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"encoding/json"
	"github.com/ywzjackal/goweb"
	"strconv"
)

const (
	ActionPrefix    = "Action"
	ActionPrefixLen = len(ActionPrefix)
	InitName        = "Init"
)

type nodeType struct {
	tp reflect.Type // reflect.type of target
	id int          // field index
}

type nodeValue struct {
	va reflect.Value // Pointer reflect.value of target
	id int           // field index
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

type controllerValue struct {
	_parent     goweb.Controller
	_selfValue  reflect.Value
	_selfType   reflect.Type
	_ctx        goweb.Context        // Realtime goweb.Context struct
	_querys     map[string]nodeValue // query parameters
	_standalone []nodeValue          // factory which need be injected after first initialized
	_stateful   []nodeValue          // factory which need be injected from session before called
	_stateless  []nodeValue          // factory which need be injected always new before called
	_type       *controllerType      // Pointer to controllerType struct
	_actions    map[string]nodeValue // methods wrap
	_init       nodeValue            // Init() function's reflect.Value Pointer
}

func (c *controllerValue) String() string {
	t := c._selfValue.Type()
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

// Context() return the context of client request
func (c *controllerValue) Context() goweb.Context {
	return c._ctx
}

// Type() return the controller type,one of FactoryTypeStandalone
// FactoryTypeStateless or FactoryTypeStatful
func (c *controllerValue) Type() goweb.LifeType {
	return c._type._type
}

func (c *controllerValue) Call(mtd string, ctx goweb.Context) ([]reflect.Value, goweb.WebError) {
	c._ctx = ctx
	act, exist := c._actions[strings.ToLower(mtd)]
	if !exist {
		act, exist = c._actions[""]
		if !exist {
			return nil, goweb.NewWebError(http.StatusMethodNotAllowed, "Action `%s` not found!", mtd)
		}
	}
	ctx_type := ctx.Request().Header.Get("Content-Type")
	if strings.Index(ctx_type, "application/json") >= 0 {
		if err := c.resolveJsonParameters(); err != nil {
			return nil, err.Append(http.StatusBadRequest, "Fail to resolve controller `%s` json parameters!", c._selfValue)
		}
	}
	if err := c.resolveUrlParameters(); err != nil {
		return nil, err.Append(http.StatusInternalServerError, "Fail to resolve controler `%s` url parameters!", c._selfValue)
	}
	if err := resolveInjections(ctx.FactoryContainer(), ctx, c._stateless); err != nil {
		return nil, err.Append(http.StatusInternalServerError, "Fail to resolve stateless injection for %s", c._selfValue)
	}
	if err := resolveInjections(ctx.FactoryContainer(), ctx, c._stateful); err != nil {
		return nil, err.Append(http.StatusInternalServerError, "Fail to resolve stateful injection for %s", c._selfValue)
	}
	rt := act.va.Call([]reflect.Value{c._selfValue})
	return rt, nil
}

type controllerType struct {
	_selfValue  reflect.Value
	_selfType   reflect.Type
	_parent     goweb.Controller
	_querys     map[string]nodeType // query parameters
	_standalone []nodeType          // factory which need be injected after first initialized
	_stateful   []nodeType          // factory which need be injected from session before called
	_stateless  []nodeType          // factory which need be injected always new before called
	_type       goweb.LifeType      // standalone or stateless or stateful
	_actions    map[string]nodeType // methods wrap
	_init       nodeType            // Init() function's reflect.Value Pointer
}

func (c *controllerType) New() *controllerValue {
	var ctl goweb.Controller = reflect.New(c._selfType).Interface().(goweb.Controller)
	rt := &controllerValue{
		_parent:     ctl,
		_selfValue:  reflect.ValueOf(ctl).Elem(),
		_selfType:   c._selfType,
		_querys:     make(map[string]nodeValue, len(c._querys)),
		_type:       c,
		_actions:    make(map[string]nodeValue, len(c._actions)),
		_standalone: make([]nodeValue, len(c._standalone)),
		_stateful:   make([]nodeValue, len(c._stateful)),
		_stateless:  make([]nodeValue, len(c._stateless)),
	}
	// initialization query(s)
	for k, n := range c._querys {
		rt._querys[k] = nodeValue{
			id: n.id,
			va: rt._selfValue.Field(n.id),
		}
	}
	// initialization injectNode
	for i, n := range rt._standalone {
		n.id = c._standalone[i].id
		n.va = c._selfValue.Field(n.id)
	}
	for i, n := range rt._stateful {
		n.id = c._stateful[i].id
		n.va = c._selfValue.Field(n.id)
	}
	for i, n := range rt._stateless {
		n.id = c._stateless[i].id
		n.va = c._selfValue.Field(n.id)
	}
	// initialization actions
	for k, n := range rt._actions {
		n.id = c._actions[k].id
		n.va = c._selfValue.Method(n.id)
	}
	// initialization Init method
	rt._init = nodeValue{
		id: c._init.id,
		va: c._selfValue.Method(c._init.id),
	}
	return rt
}
func newControlerType(ctli goweb.Controller, fac goweb.FactoryContainer) *controllerType {
	var (
		rtp reflect.Type    = reflect.TypeOf(ctli)
		rva reflect.Value   = reflect.ValueOf(ctli)
		ctl *controllerType = &controllerType{
			_selfValue: rva,
			_selfType:  reflect.TypeOf(ctli).Elem(),
			_querys:    make(map[string]nodeType),
			_actions:   make(map[string]nodeType),
			_parent:    ctli,
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
	if ctl._type == goweb.LifeTypeError {
		panic("goweb.Controller need extend from one of interface ControllerStandalone/ControllerStateful/ControllerStateless")
	}
	if err := ctl.initSubFields(rva, fac); err != nil {
		panic(err.ErrorAll())
	}
	for i := 0; i < rtp.NumMethod(); i++ {
		mtd := rtp.Method(i)
		if isActionMethod(&mtd) != nil {
			continue
		}
		if mtd.Name == InitName {
			ctl._init = nodeType{
				id: i,
				tp: rtp.Field(i).Type,
			}
		} else {
			name := strings.ToLower(mtd.Name[ActionPrefixLen:])
			ctl._actions[name] = nodeType{id: i, tp: mtd.Type}
			goweb.Log.Printf("INIT goweb.Controller `%s` -> `%s` (%s)", rtp, name, goweb.LifeTypeName[ctli.Type()])
		}
	}
	return ctl
}

func (ctl *controllerType) initSubFields(rva reflect.Value, fac goweb.FactoryContainer) goweb.WebError {
	for i := 0; i < rva.NumField(); i++ {
		stfd := rva.Type().Field(i) // struct field
		fdva := rva.Field(i)        // field value
		if !fdva.CanSet() {
			continue
		}
		switch stfd.Type.Kind() {
		case reflect.Int, reflect.String, reflect.Float32, reflect.Bool, reflect.Slice:
			ctl._querys[strings.ToLower(stfd.Name)] = nodeType{
				id: i,
				tp: stfd.Type,
			}
		case reflect.Ptr:
			if isTypeLookupAble(stfd.Type) != nil {
				break
			}
			switch factoryType(stfd.Type) {
			case goweb.LifeTypeStandalone:
				ctl._standalone = append(ctl._stateful, nodeType{
					id: i,
					tp: stfd.Type,
				})
				_va, err := fac.Lookup(stfd.Type, nil)
				if err != nil {
					return err.Append(500, "Fail to inject `%s` in `%s`", stfd.Type, ctl._selfValue.Type())
				}
				fdva.Set(_va)
			case goweb.LifeTypeStateful:
				ctl._stateful = append(ctl._stateful, nodeType{
					id: i,
					tp: stfd.Type,
				})
			case goweb.LifeTypeStateless:
				ctl._stateless = append(ctl._stateful, nodeType{
					id: i,
					tp: stfd.Type,
				})
			default:
				return goweb.NewWebError(500, "Factory `%s` type is not be specified", stfd.Type)
			}
		case reflect.Struct:
			ctl.initSubFields(fdva, fac)
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

func resolveInjections(factorys goweb.FactoryContainer, ctx goweb.Context, nodes []nodeValue) goweb.WebError {
	for _, node := range nodes {
		v, err := factorys.Lookup(node.va.Type(), ctx)
		if err != nil {
			return err.Append(500, "Fail to inject `%s`", node.va.Type())
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

func (c *controllerValue) resolveJsonParameters() goweb.WebError {
	de := json.NewDecoder(c._ctx.Request().Body)
	err := de.Decode(c._parent)
	if err != nil {
		return goweb.NewWebError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func (c *controllerValue) resolveUrlParameters() goweb.WebError {
	req := c._ctx.Request()
	if err := req.ParseForm(); err != nil {
		return goweb.NewWebError(500, "Fail to ParseForm with path:%s,%s", req.URL.String(), err.Error())
	}
	//	for key, node := range c._querys {
	for pn, pv := range req.Form {
		key := strings.ToLower(pn)
		//		strs := req.Form[key]
		strs := pv
		node, ok := c._querys[key]
		if !ok {
			continue
		}
		switch node.va.Kind() {
		case reflect.String:
			if len(strs) == 0 {
				node.va.SetString("")
				break
			}
			node.va.SetString(strs[0])
		case reflect.Bool:
			if len(strs) == 0 {
				node.va.SetBool(false)
				break
			}
			node.va.SetBool(ParseBool(strs[0]))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if len(strs) == 0 {
				node.va.SetInt(0)
				break
			}
			num, err := strconv.ParseInt(strs[0], 10, 0)
			if err != nil {
				goweb.Err.Printf("Fail to convent parameters!\r\nField `%s`(int) can not set by '%s'", node.va.Type().Name(), req.Form.Get(key))
				continue
			}
			node.va.SetInt(num)
		case reflect.Float32, reflect.Float64:
			if len(strs) == 0 {
				node.va.SetFloat(0.0)
				break
			}
			f, err := strconv.ParseFloat(strs[0], 0)
			if err != nil {
				goweb.Err.Printf("Fail to convent parameters!\r\nField `%s`(float) can not set by '%s'", node.va.Type().Name(), req.Form.Get(key))
				continue
			}
			node.va.SetFloat(f)
		case reflect.Slice:
			targetType := node.va.Type().Elem()
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
			return goweb.NewWebError(500, "Unresolveable url parameter type `%s`", node.va.Type())
		}
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
