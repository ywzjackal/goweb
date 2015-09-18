package controller

import (
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

var (
	factoryTypesByName = map[string]goweb.LifeType{
		"FactoryStandalone": goweb.LifeTypeStandalone,
		"FactoryStateful":   goweb.LifeTypeStateful,
		"FactoryStateless":  goweb.LifeTypeStateless,
	}
)

type controller struct {
	_ctx goweb.Context
}

func (c *controller) Init() {

}

func (c *controller) Context() goweb.Context {
	return c._ctx
}

type ControllerStandalone struct {
	controller
}

func (c *ControllerStandalone) Type() goweb.LifeType {
	return goweb.LifeTypeStandalone
}

type ControllerStateless struct {
	controller
}

func (c *ControllerStateless) Type() goweb.LifeType {
	return goweb.LifeTypeStateless
}

type ControllerStateful struct {
	controller
}

func (c *ControllerStateful) Type() goweb.LifeType {
	return goweb.LifeTypeStateful
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

func (c *ctlCallable) resolveJsonParameters() goweb.WebError {
	de := json.NewDecoder(c._ctx.Request().Body)
	err := de.Decode(c._interface)
	if err != nil {
		return goweb.NewWebError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func (c *ctlCallable) resolveUrlParameters() goweb.WebError {
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
	s := ""
	for rt.Kind() == reflect.Ptr {
		s = s + "*"
		rt = rt.Elem()
	}
	return s + rt.Kind().String()
}

func elemOfVal(val reflect.Value) reflect.Value {
	for val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	return val
}

func elemOfTyp(typ reflect.Type) reflect.Type {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	return typ
}
