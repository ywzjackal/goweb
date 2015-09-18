package controller

import (
	"github.com/ywzjackal/goweb"
	"net/http"
	"reflect"
)

type context struct {
	goweb.Context
	_c goweb.FactoryContainer
}

func (c *context) Request() *http.Request {
	return nil
}
func (c *context) ResponseWriter() http.ResponseWriter {
	return nil
}
func (c *context) FactoryContainer() goweb.FactoryContainer {
	return c._c
}
func (c *context) Session() goweb.Session {
	return nil
}
func (c *context) Error() goweb.WebError {
	return nil
}
func (c *context) SetError(goweb.WebError) {

}

// controllerContainer is buildin default controller container
type controllerContainer struct {
	goweb.ControllerContainer
	factorys goweb.FactoryContainer
	ctls     map[string]goweb.ControllerSchema   // store all controller scheme
	sds      map[string]goweb.ControllerCallAble // store standalone controllers
}

func NewControllerContainer(f goweb.FactoryContainer) goweb.ControllerContainer {
	c := &controllerContainer{
		factorys: f,
		ctls:     make(map[string]goweb.ControllerSchema),
		sds:      make(map[string]goweb.ControllerCallAble),
	}
	return c
}

func (c *controllerContainer) Register(prefix string, ctl goweb.Controller) {
	_, exist := c.ctls[prefix]
	if exist {
		panic("URL Prefix:" + prefix + " register duplicated")
	}
	sch := &schema{}
	sch.Init(ctl)
	c.ctls[prefix] = sch
	if ctl.Type() == goweb.LifeTypeStandalone {
		c.sds[prefix] = sch.NewCallAble()
	}
}

func (c *controllerContainer) Get(prefix string, ctx goweb.Context) (goweb.ControllerCallAble, goweb.WebError) {
	var (
		sch goweb.ControllerSchema = nil
		ok  bool                   = false
	)
	sch, ok = c.ctls[prefix]
	if !ok || sch == nil {
		return nil, goweb.NewWebError(404, "goweb.Controller `%s` not found!", prefix)
	}
	switch sch.Type() {
	case goweb.LifeTypeStandalone: //no need break;
		return c.sds[prefix], nil
	case goweb.LifeTypeStateless:
		//		ctl = reflect.New(reflect.TypeOf(ctl).Elem()).Interface().(goweb.Controller)
		//		initController(ctl, ctx)
		return sch.NewCallAble(), nil
	case goweb.LifeTypeStateful:
		mem := ctx.Session().MemMap()
		itfs, ok := mem["__ctl_"+prefix]
		if !ok {
			//			ctl = reflect.New(reflect.TypeOf(ctl).Elem()).Interface().(goweb.Controller)
			//			initController(ctl, ctx)
			cv := sch.NewCallAble()
			mem["__ctl_"+prefix] = cv
			return cv, nil
		} else {
			cv, ok := itfs.(goweb.ControllerCallAble)
			if !ok {
				return nil, goweb.NewWebError(500, "Fail to convert session stateful interface to controller!")
			}
			return cv, nil
		}
	default:
		return nil, goweb.NewWebError(500, "goweb.Controller Life Status Undefined!")
	}
}

func isInterfaceController(itfs interface{}) goweb.WebError {
	var (
		t = reflect.TypeOf(itfs)
	)
	_, ok := itfs.(*schema)
	if !ok {
		return goweb.NewWebError(500, "`%s` is not based on goweb.goweb.Controller!", t)
	}
	if t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct {
		return nil
	}
	return goweb.NewWebError(500, "`%s` is not a pointer of struct!", reflectType(t))
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
