package goweb

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

func reflectType(rt reflect.Type) string {
	typstr := ""
	for rt.Kind() == reflect.Ptr {
		typstr = typstr + "*"
		rt = rt.Elem()
	}
	return typstr + rt.Kind().String()
}

func isActionMethod(method *reflect.Method) WebError {
	if !strings.HasPrefix(method.Name, ActionPrefix) {
		return NewWebError(500, "func %s doesn't have prefix '%s'", method.Name, ActionPrefix)
	}
	if method.Type.NumIn() != 1 {
		err := NewWebError(500, "Action func %s need function without parameters in! got %d", method.Name, method.Type.NumIn()-1)
		Err.Print(err.Error())
		return err
	}
	if method.Type.NumOut() == 0 || method.Type.Out(0).Kind() != reflect.String {
		err := NewWebError(500, "Action func %s need function with at least on parameters out, the first parameters out must be string to specify which view do you need,like 'json','txt','html' etc.", method.Name)
		Err.Print(err.Error())
		return err
	}
	return nil
}

func isInterfaceController(itfs interface{}) WebError {
	var (
		t = reflect.TypeOf(itfs)
	)
	_, ok := itfs.(*controller)
	if !ok {
		return NewWebError(500, "`%s` is not based on goweb.Controller!", t)
	}
	if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
		return nil
	}
	return NewWebError(500, "`%s` is not a pointer of struct!", reflectType(t))
}

func isTypeLookupAble(rt reflect.Type) WebError {
	if rt.Kind() == reflect.Interface {
		return nil
	}
	if rt.Kind() == reflect.Ptr && rt.Elem().Kind() == reflect.Struct {
		return nil
	}
	return NewWebError(500, "`%s(%s)` is not a interface or *struct", rt, reflectType(rt))
}

func isTypeRegisterAble(rt reflect.Type) WebError {
	if rt.Kind() == reflect.Ptr && rt.Elem().Kind() == reflect.Struct {
		return nil
	}
	return NewWebError(500, "`%s(%s)` is not a *struct", rt, reflectType(rt))
}

func lookupAndInjectFactories(typs []reflect.Type, ctx Context) ([]reflect.Value, WebError) {
	rts := make([]reflect.Value, len(typs))
	for i, typ := range typs {
		_v, _e := ctx.FactoryContainer().Lookup(typ, ctx)
		if _e != nil {
			return nil, _e.Append(500, "Fail to inject `%s`", typ)
		}
		rts[i] = _v
	}
	return rts, nil
}

func lookupAndInjectFromContext(paramType reflect.Type, context Context) (reflect.Value, WebError) {
	var (
		parameterValuePointer = reflect.Value{}
		req                   = context.Request().Form
	)
	if paramType.Kind() != reflect.Ptr {
		return parameterValuePointer,
			NewWebError(1, "Controller Method's first Parameter must a pointer of Parameters")
	}
	pt := paramType.Elem()
	fieldNum := pt.NumField()
	//	Log.Printf("Paramter `%s` has %d Field(s).", pt.Name(), fieldNum)
	// foreach Parameters field
	parameterValuePointer = reflect.New(pt)
	parameterValue := parameterValuePointer.Elem()
	for i := 0; i < fieldNum; i++ {
		field := pt.Field(i)
		value := req.Get(field.Name)
		if !parameterValue.Field(i).IsValid() {
			Err.Printf("Fail convent url.values to ParametersInterface!\r\nField `%s` is invalid !", field.Name)
			continue
		}
		if !parameterValue.Field(i).CanSet() {
			//Err.Printf("Fail convent url.values to ParametersInterface!\r\nField `%s` can not be set!", field.Name)
			continue
		}
		switch field.Type.Kind() {
		case reflect.String:
			parameterValue.Field(i).SetString(value)
		case reflect.Bool:
			b, err := strconv.ParseBool(value)
			if err != nil {
				Err.Printf("Fail to convent parameters!\r\nField `%s`(bool) can not set by '%s'", field.Name, value)
				continue
			}
			parameterValue.Field(i).SetBool(b)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			num, err := strconv.ParseInt(value, 10, 0)
			if err != nil {
				Err.Printf("Fail to convent parameters!\r\nField `%s`(int) can not set by '%s'", field.Name, value)
				continue
			}
			parameterValue.Field(i).SetInt(num)
		case reflect.Float32, reflect.Float64:
			f, err := strconv.ParseFloat(value, 0)
			if err != nil {
				Err.Printf("Fail to convent parameters!\r\nField `%s`(float) can not set by '%s'", field.Name, value)
				parameterValue.Field(i).SetFloat(0.0)
				continue
			}
			parameterValue.Field(i).SetFloat(f)
		case reflect.Struct:
			continue
		case reflect.Slice:
			targetType := field.Type.Elem()
			stringValues := req[field.Name]
			lens := len(stringValues)
			values := reflect.MakeSlice(reflect.SliceOf(targetType), lens, lens)
			switch targetType.Kind() {
			case reflect.String:
				values = reflect.ValueOf(stringValues)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				for j := 0; j < lens; j++ {
					v := values.Index(j)
					intValue, _ := strconv.ParseInt(stringValues[j], 10, 0)
					v.SetInt(intValue)
				}
			case reflect.Float32, reflect.Float64:
				for j := 0; j < lens; j++ {
					v := values.Index(j)
					floatValue, _ := strconv.ParseFloat(stringValues[j], 0)
					v.SetFloat(floatValue)
				}
			case reflect.Bool:
				for j := 0; j < lens; j++ {
					v := values.Index(j)
					boolValue, _ := strconv.ParseBool(stringValues[j])
					v.SetBool(boolValue)
				}
			}
			parameterValue.Field(i).Set(values)
		case reflect.Interface, reflect.Ptr:
			v, e := context.FactoryContainer().Lookup(field.Type, context)
			if e != nil {
				return parameterValuePointer, e
			}
			parameterValue.Field(i).Set(v)
		}
	}
	return parameterValuePointer, nil
}

func generateSessionIdByRequest(req *http.Request) string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}

func render(rets []reflect.Value, c Controller) WebError {
	if len(rets) == 0 {
		return NewWebError(1, "Controller Action need return a ViewType like `html`,`json`.")
	}
	viewType, ok := rets[0].Interface().(string)
	if !ok {
		return NewWebError(1, "Controller Action need return a ViewType of string! but got `%s`", rets[0].Type())
	}
	view, isexist := views[viewType]
	if !isexist {
		return NewWebError(1, "Unknow ViewType :%s", viewType)
	}
	interfaces := make([]interface{}, len(rets)-1, len(rets)-1)
	for i, ret := range rets[1:] {
		interfaces[i] = ret.Interface()
	}
	err := view.Render(c, interfaces...)
	if err != nil {
		return err.Append(500, "Fail to render view %s, data:%+v", viewType, interfaces)
	}
	return nil
}

func resolveInjections(factorys FactoryContainer, ctx Context, nodes []injectNode) WebError {
	for _, node := range nodes {
		v, err := factorys.Lookup(node.tp, ctx)
		if err != nil {
			return err.Append(500, "Fail to inject `%s`", node.tp)
		}
		if !v.IsValid() {
			return NewWebError(500, "inject invalid value to `%s`", node.va)
		}
		if v.Kind() != reflect.Ptr {
			return NewWebError(500, "inject invalid type of %s, need Ptr", v.Kind())
		}
		node.va.Set(v)
	}
	return nil
}

func resolveUrlParameters(c *controller, target *reflect.Value) WebError {
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
