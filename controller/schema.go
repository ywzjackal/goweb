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
	Target      goweb.Controller         // Target must be set before register to container
	_tValue     reflect.Value            // target(user defined controller struct) reflect.Value
	_tType      reflect.Type             // target(user defined controller struct) reflect.Type
	_query      map[string]nodeType      // query parameters
	_standalone []nodeType               // factory which need be injected after first initialized
	_stateful   []nodeType               // factory which need be injected from session before called
	_stateless  []nodeType               // factory which need be injected always new before called
	_actions    map[string]nodeType      // methods wrap
	_init       nodeType                 // Init() function's reflect.Value Pointer
	_master     goweb.ControllerCallAble // Master Value
	_lft		goweb.LifeType
}

func (c *schema) NewCallAble() goweb.ControllerCallAble {
	var nc goweb.Controller = reflect.New(c._tType).Interface().(goweb.Controller)
	_selfValue := reflect.ValueOf(nc)
	for reflect.Ptr == _selfValue.Kind() {
		_selfValue = _selfValue.Elem()
	}
	rt := &ctlCallable{
		_interface:  nc,
		_selfValue:  _selfValue,
		_selfType:   c._tType,
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
			va: rt._selfValue.Field(n.id),
		}
	}
	// initialization injectNode
	for i, n := range rt._standalone {
		n.id = c._standalone[i].id
		n.va = c._tValue.Field(n.id)
	}
	for i, n := range rt._stateful {
		n.id = c._stateful[i].id
		n.va = c._tValue.Field(n.id)
	}
	for i, n := range rt._stateless {
		n.id = c._stateless[i].id
		n.va = c._tValue.Field(n.id)
	}
	// initialization actions
	for k, n := range rt._actions {
		n.id = c._actions[k].id
		n.va = c._tValue.Method(n.id)
	}
	return rt
}

func (c *schema) Init(ctl goweb.Controller) {
	c.Target = ctl
	rva := elemOfVal(reflect.ValueOf(ctl))
	rtp := elemOfTyp(reflect.TypeOf(ctl))
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
	c._master = c.NewCallAble()
}

func (c *schema) Type() goweb.LifeType  {
	return c._lft
}

func (c *schema) initActions() {
	rtp := c._tType
	for i := 0; i < rtp.NumMethod(); i++ {
		mtd := rtp.Method(i)
		if isActionMethod(&mtd) != nil {
			continue
		}
		if mtd.Name == InitName {
			c._init = nodeType{
				id: i,
				tp: rtp.Field(i).Type,
			}
		} else {
			name := strings.ToLower(mtd.Name[ActionPrefixLen:])
			c._actions[name] = nodeType{id: i, tp: mtd.Type}
			goweb.Log.Printf("INIT goweb.Controller `%s` -> `%s`", rtp, name)
		}
	}
}

func (c *schema) initSubFields() {
	rva := c._tValue
	for i := 0; i < rva.NumField(); i++ {
		stfd := rva.Type().Field(i) // struct field
		fdva := rva.Field(i)        // field value
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
