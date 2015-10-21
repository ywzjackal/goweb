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

type Controller struct {
	_ctx goweb.Context
}

func (c *Controller) SetContext(ctx goweb.Context) {
	c._ctx = ctx
}

func (c *Controller) Context() goweb.Context {
	return c._ctx
}

type Standalone struct{}

func (c *Standalone) Type() goweb.LifeType {
	return goweb.LifeTypeStandalone
}

type Stateless struct{}

func (c *Stateless) Type() goweb.LifeType {
	return goweb.LifeTypeStateless
}

type Stateful struct{}

func (c *Stateful) Type() goweb.LifeType {
	return goweb.LifeTypeStateful
}

func injectionStandalone(container goweb.FactoryContainer, nodes []goweb.InjectNode) goweb.WebError {
	for _, node := range nodes {
		v, err := container.LookupStandalone(node.Name)
		if err != nil {
			return err.Append("look up (%s) fail", node.Name)
		}
		if v.ReflectValue().Type().AssignableTo(node.Value.Type()) {
			node.Value.Set(v.ReflectValue())
		} else {
			return goweb.NewWebError(http.StatusInternalServerError,
				"can not inject %s by %s with name %s", node.Value.Type(), v.ReflectValue().Type(), node.Name)
		}
	}
	return nil
}

func injectionStateless(container goweb.FactoryContainer, nodes []goweb.InjectNode) goweb.WebError {
	for _, node := range nodes {
		v, err := container.LookupStateless(node.Name)
		if err != nil {
			return err.Append("look up fail")
		}
		if v.ReflectValue().Type().AssignableTo(node.Value.Type()) {
			node.Value.Set(v.ReflectValue())
		} else {
			return goweb.NewWebError(http.StatusInternalServerError,
				"can not inject %s by %s with name %s", node.Value.Type(), v.ReflectValue().Type(), node.Name)
		}
	}
	return nil
}

func injectionStateful(container goweb.FactoryContainer, nodes []goweb.InjectNode, state goweb.InjectGetterSetter) goweb.WebError {
	for _, node := range nodes {
		v, err := container.LookupStateful(node.Name, state)
		if err != nil {
			return err.Append("look up(%s) fail", node.Name)
		}
		if v.ReflectValue().Type().AssignableTo(node.Value.Type()) {
			node.Value.Set(v.ReflectValue())
		} else {
			return goweb.NewWebError(http.StatusInternalServerError,
				"can not inject %s by %s with name %s", node.Value.Type(), v.ReflectValue().Type(), node.Name)
		}
	}
	return nil
}

func (c *ctlCallable) resolveJsonParameters(ctx goweb.Context) goweb.WebError {
	de := json.NewDecoder(ctx.Request().Body)
	err := de.Decode(c._interface)
	if err != nil {
		return goweb.NewWebError(http.StatusBadRequest, err.Error())
	}
	return nil
}

func (c *ctlCallable) resolveUrlParameters(ctx goweb.Context) goweb.WebError {
	req := ctx.Request()
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
		switch node.Kind() {
		case reflect.String:
			if len(strs) == 0 {
				node.SetString("")
				break
			}
			node.SetString(strs[0])
		case reflect.Bool:
			if len(strs) == 0 {
				node.SetBool(false)
				break
			}
			node.SetBool(ParseBool(strs[0]))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if len(strs) == 0 {
				node.SetInt(0)
				break
			}
			num, err := strconv.ParseInt(strs[0], 10, 0)
			if err != nil {
				goweb.Err.Printf("Fail to convent parameters!\r\nField `%s`(int) can not set by '%s'", node.Type().Name(), req.Form.Get(key))
				continue
			}
			node.SetInt(num)
		case reflect.Float32, reflect.Float64:
			if len(strs) == 0 {
				node.SetFloat(0.0)
				break
			}
			f, err := strconv.ParseFloat(strs[0], 0)
			if err != nil {
				goweb.Err.Printf("Fail to convent parameters!\r\nField `%s`(float) can not set by '%s'", node.Type().Name(), req.Form.Get(key))
				continue
			}
			node.SetFloat(f)
		case reflect.Slice:
			targetType := node.Type().Elem()
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
			node.Set(values)
		default:
			return goweb.NewWebError(500, "Unresolveable url parameter type `%s`", node.Type())
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
