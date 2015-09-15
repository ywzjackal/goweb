package controller

import (
	"reflect"
	"strconv"

	"github.com/ywzjackal/goweb"
)

func NewControllerContainer(f goweb.FactoryContainer) goweb.ControllerContainer {
	c := &controllerContainer{
		factorys: f,
		ctls:     make(map[string]goweb.Controller),
	}
	return c
}

// controllerContainer is buildin default controller container
type controllerContainer struct {
	goweb.ControllerContainer
	ctls     map[string]goweb.Controller
	factorys goweb.FactoryContainer
}

func (c *controllerContainer) Register(prefix string, ctl goweb.Controller) {
	_, exist := c.ctls[prefix]
	if exist {
		panic("URL Prefix:" + prefix + " register duplicated")
	}
	initController(ctl, c.factorys)
	c.ctls[prefix] = ctl
}

func (c *controllerContainer) Get(prefix string, ctx goweb.Context) (goweb.Controller, goweb.WebError) {
	var (
		ctl goweb.Controller = nil
		ok  bool             = false
	)
	ctl, ok = c.ctls[prefix]
	if !ok || ctl == nil {
		return nil, goweb.NewWebError(404, "goweb.Controller `%s` not found!", prefix)
	}
	switch ctl.Type() {
	case goweb.LifeTypeStandalone:
	case goweb.LifeTypeStateless:
		ctl = reflect.New(reflect.TypeOf(ctl).Elem()).Interface().(goweb.Controller)
		initController(ctl, ctx.FactoryContainer())
	case goweb.LifeTypeStateful:
		mem := ctx.Session().MemMap()
		itfs, ok := mem["__ctl_"+prefix]
		if !ok {
			ctl = reflect.New(reflect.TypeOf(ctl).Elem()).Interface().(goweb.Controller)
			initController(ctl, ctx.FactoryContainer())
			mem["__ctl_"+prefix] = ctl
		} else {
			ctl, ok = itfs.(goweb.Controller)
			if !ok {
				return nil, goweb.NewWebError(500, "Fail to convert session stateful interface to controller!")
			}
		}
	default:
		return nil, goweb.NewWebError(500, "goweb.Controller Life Status Undefined!")
	}
	return ctl, nil
}

func isInterfaceController(itfs interface{}) goweb.WebError {
	var (
		t = reflect.TypeOf(itfs)
	)
	_, ok := itfs.(*controller)
	if !ok {
		return goweb.NewWebError(500, "`%s` is not based on goweb.goweb.Controller!", t)
	}
	if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
		return nil
	}
	return goweb.NewWebError(500, "`%s` is not a pointer of struct!", reflectType(t))
}

func resolveJsonParameters(c *controller, target *reflect.Value) goweb.WebError {
	return nil
}

func resolveUrlParameters(c *controller, target *reflect.Value) goweb.WebError {
	req := c._ctx.Request()
	if err := req.ParseForm(); err != nil {
		return goweb.NewWebError(500, "Fail to ParseForm with path:%s,%s", req.URL.String(), err.Error())
	}
	for key, node := range c._querys {
		strs := req.Form[key]
		switch node.tp.Kind() {
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
				goweb.Err.Printf("Fail to convent parameters!\r\nField `%s`(int) can not set by '%s'", node.tp.Name(), req.Form.Get(key))
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
				goweb.Err.Printf("Fail to convent parameters!\r\nField `%s`(float) can not set by '%s'", node.tp.Name(), req.Form.Get(key))
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
			return goweb.NewWebError(500, "Unresolveable url parameter type `%s`", node.tp)
		}
	}
	return nil
}

// ParseBool returns the boolean value represented by the string.
// It accepts 1, t, T, TRUE, true, True, 0, f, F, FALSE, false, False.
// Any other value returns an error.
func ParseBool(str string) (value bool) {
	goweb.Log.Print(str)
	switch str {
	case "1", "t", "T", "true", "TRUE", "True", "on", "ON", "On", "O", "o":
		return true
	default:
		return false
	}
}
