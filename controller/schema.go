package controller

import (
	"github.com/ywzjackal/goweb"
	"reflect"
	"strings"
)

type nodeType struct {
	tp reflect.Type  // reflect.type of target
	dv reflect.Value // default value for this type
	id int           // field index
}

type schema struct {
	Target      goweb.Controller    // Target must be set before register to container
	_tValue     reflect.Value       // target(user defined controller struct) reflect.Value
	_tType      reflect.Type        // target(user defined controller struct) reflect.Type
	_query      map[string]nodeType // query parameters
	_standalone []nodeType          // factory which need be injected after first initialized
	_stateful   []nodeType          // factory which need be injected from session before called
	_stateless  []nodeType          // factory which need be injected always new before called
	_actions    map[string]nodeType // methods wrap
	_init       nodeType            // Init() function's reflect.Value Pointer
	_lft        goweb.LifeType
}

func (c *schema) NewCallAble() goweb.ControllerCallAble {
	var nc goweb.Controller = reflect.New(c._tType.Elem()).Interface().(goweb.Controller)
	_selfValue := reflect.ValueOf(nc)
	rt := &ctlCallable{
		_interface:  nc,
		_selfValue:  _selfValue,
		_querys:     make(map[string]nodeValue, len(c._query)),
		_actions:    make(map[string]nodeValue, len(c._actions)),
		_standalone: make([]nodeValue, len(c._standalone)),
		_stateful:   make([]nodeValue, len(c._stateful)),
		_stateless:  make([]nodeValue, len(c._stateless)),
	}
	// initialization query(s)
	for k, n := range c._query {
		rt._querys[k] = nodeValue{
			id: n.id,
			va: rt._selfValue.Elem().Field(n.id),
		}
	}
	// initialization injectNode
	for i, n := range c._standalone {
		rt._standalone[i] = nodeValue{
			id: n.id,
			va: rt._selfValue.Elem().Field(n.id),
		}
	}
	for i, n := range c._stateful {
		rt._stateful[i] = nodeValue{
			id: n.id,
			va: rt._selfValue.Elem().Field(n.id),
		}
	}
	for i, n := range c._stateless {
		rt._stateless[i] = nodeValue{
			id: n.id,
			va: rt._selfValue.Elem().Field(n.id),
		}
	}
	// initialization actions
	for k, n := range c._actions {
		rt._actions[k] = nodeValue{
			id: c._actions[k].id,
			va: rt._selfValue.Method(n.id),
		}
	}
	return rt
}

func (c *schema) Init(ctl goweb.Controller) {
	c.Target = ctl
	rva := reflect.ValueOf(ctl)
	rtp := reflect.TypeOf(ctl)
	c._tValue = rva
	c._tType = rtp
	c._query = make(map[string]nodeType)
	c._actions = make(map[string]nodeType)
	c._lft = ctl.Type()
	for rva.Kind() == reflect.Ptr {
		rva = rva.Elem()
	}
	c.initSubFields()
	c.initActions()
}

func (c *schema) Type() goweb.LifeType {
	return c._lft
}

func (c *schema) initActions() {
	rtp := c._tType
	for i := 0; i < rtp.NumMethod(); i++ {
		mtd := rtp.Method(i)
		if !isActionMethod(&mtd) {
			continue
		}
		name := strings.ToUpper(mtd.Name[ActionPrefixLen:])
		c._actions[name] = nodeType{id: i, tp: mtd.Type}
	}
	if goweb.Debug {
		str := ""
		for k, _ := range c._actions {
			str += "[" + strings.ToUpper(k) + "] "
		}
		goweb.Log.Printf("init `%s` actions(%d) %s", rtp.Elem().Name(), len(c._actions), str)
	}
}

func (c *schema) initSubFields() {
	rva := c._tValue
	for i := 0; i < rva.Elem().NumField(); i++ {
		stfd := rva.Elem().Type().Field(i) // struct field
		fdva := rva.Elem().Field(i)        // field value
		if !fdva.CanSet() {
			continue
		}
		switch stfd.Type.Kind() {
		case reflect.Int, reflect.String, reflect.Float32, reflect.Bool, reflect.Slice:
			c._query[strings.ToLower(stfd.Name)] = nodeType{
				id: i,
				tp: stfd.Type,
			}
		case reflect.Ptr:
			if isTypeLookupAble(stfd.Type) != nil {
				break
			}
			switch factoryType(stfd.Type) {
			case goweb.LifeTypeStandalone:
				c._standalone = append(c._stateful, nodeType{
					id: i,
					tp: stfd.Type,
				})
			case goweb.LifeTypeStateful:
				c._stateful = append(c._stateful, nodeType{
					id: i,
					tp: stfd.Type,
				})
			case goweb.LifeTypeStateless:
				c._stateless = append(c._stateful, nodeType{
					id: i,
					tp: stfd.Type,
				})
			default:
				panic(goweb.NewWebError(500, "Factory `%s` type is not be specified", stfd.Type).ErrorAll())
			}
		case reflect.Struct:
			//			ctl.initSubFields(fdva)
		}
	}
}

func isActionMethod(method *reflect.Method) bool {
	if !strings.HasPrefix(method.Name, ActionPrefix) {
		if goweb.Debug {
			//goweb.Log.Printf("`%s` is not a action because no prefix with `%s`", method.Name, ActionPrefix)
		}
		return false
	}
	if method.Type.NumIn() != 1 {
		if goweb.Debug {
			goweb.Log.Printf("`%s` is not a action because number in is %d != 0", method.Name, method.Type.NumIn() - 1)
		}
		return false
	}
	if method.Type.NumOut() != 0 {
		if goweb.Debug {
			goweb.Log.Printf("`%s` is not a action because number out is %d != 0", method.Name, method.Type.NumOut())
		}
		return false
	}
	return true
}
