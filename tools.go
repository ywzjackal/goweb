package goweb

import (
	"crypto/sha1"
	"encoding/base64"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func paramtersFromRequestUrl(paramType reflect.Type, context Context) reflect.Value {
	var (
		parameterValuePointer = reflect.Value{}
		req                   = context.Request().Form
	)
	if paramType.Kind() != reflect.Ptr {
		Err.Printf("Controller Method's first Parameter must a pointer of Parameters")
		return parameterValuePointer
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
		case reflect.Interface, reflect.Ptr:
			v, e := context.FactoryContainer().Lookup(field.Type, context)
			if e != nil {
				continue
			}
			parameterValue.Field(i).Set(v)
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
		}
	}
	return parameterValuePointer
}

func generateSessionIdByRequest(req *http.Request) string {
	now := time.Now()
	sha := sha1.New()
	source := []byte(now.String() + req.Host + req.RemoteAddr)
	sum := sha.Sum(source)
	return base64.StdEncoding.EncodeToString(sum)
}

func initActionWrap(index int, method *reflect.Method, caller reflect.Value) *actionWrap {
	aw := &actionWrap{
		index:          index,
		actionName:     method.Name,
		name:           strings.ToLower(method.Name),
		method:         method,
		parameters:     make([]reflect.Value, method.Type.NumIn()),
		parameterTypes: make([]reflect.Type, method.Type.NumIn()),
	}
	for i := 0; i < method.Type.NumIn(); i++ {
		t := method.Type.In(i)
		aw.parameterTypes[i] = t
		//		Log.Printf("%+v", t)
	}
	aw.parameters[0] = caller
	return aw
}

func initControllerWrap(name string, c Controller) *controllerWrap {
	v := reflect.ValueOf(c)
	t := reflect.TypeOf(c)
	if name == "" {
		name = t.Elem().Name()
	} else {
		name = ControllerPrefix + name
	}
	wrap := &controllerWrap{
		controllerName: t.Name(),
		name:           strings.ToLower(name),
		fullName:       t.PkgPath() + t.Name(),
		actions:        make(map[string]*actionWrap),
	}
	for i := 0; i < t.NumMethod(); i++ {
		m := t.Method(i)
		if strings.HasPrefix(m.Name, ActionPrefix) {
			aw := initActionWrap(i, &m, v)
			wrap.actions[aw.name] = aw
			Log.Printf("Init `%s` : `%s`", wrap.name, m.Name)
		} else {
			Log.Printf("Igno `%s` : `%s`", wrap.name, m.Name)
		}
	}
	return wrap
}
