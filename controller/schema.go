package controller

import (
	"github.com/ywzjackal/goweb"
	"net/http"
	"reflect"
	"strings"
)

const (
	InjectTagName = "inject"
)

type fieldType struct {
	name string // factory name
	id   []int  // field index
}

type schema struct {
	Target         goweb.Controller     // Target must be set before register to container
	_tValue        reflect.Value        // target(user defined controller struct) reflect.Value
	_tType         reflect.Type         // target(user defined controller struct) reflect.Type
	_query         map[string]fieldType // query parameters
	_standalone    []fieldType          // factory which need be injected after first initialized
	_stateful      []fieldType          // factory which need be injected from session before called
	_stateless     []fieldType          // factory which need be injected always new before called
	_actions       map[string]int       // methods wrap
	_init          int                  // Init() function's reflect.Value Pointer
	_lft           goweb.LifeType
	_hasPreAction  bool
	_hasPostAction bool
}

func (c *schema) NewCallAble() goweb.ControllerCallAble {
	var nc goweb.Controller = reflect.New(c._tType.Elem()).Interface().(goweb.Controller)
	_selfValue := reflect.ValueOf(nc)
	rt := &ctlCallable{
		_interface:  nc,
		_selfValue:  _selfValue,
		_querys:     make(map[string]reflect.Value, len(c._query)),
		_actions:    make(map[string]reflect.Value, len(c._actions)),
		_standalone: make([]goweb.InjectNode, len(c._standalone)),
		_stateful:   make([]goweb.InjectNode, len(c._stateful)),
		_stateless:  make([]goweb.InjectNode, len(c._stateless)),
	}
	// initialization query(s)
	for k, n := range c._query {
		rt._querys[k] = rt._selfValue.Elem().FieldByIndex(n.id)
	}
	// initialization injectNode
	for i, n := range c._standalone {
		rt._standalone[i] = goweb.InjectNode{
			Name:  n.name,
			Value: rt._selfValue.Elem().FieldByIndex(n.id),
		}
	}
	for i, n := range c._stateful {
		rt._stateful[i] = goweb.InjectNode{
			Name:  n.name,
			Value: rt._selfValue.Elem().FieldByIndex(n.id),
		}
	}
	for i, n := range c._stateless {
		rt._stateless[i] = goweb.InjectNode{
			Name:  n.name,
			Value: rt._selfValue.Elem().FieldByIndex(n.id),
		}
	}
	// initialization actions
	for k, n := range c._actions {
		rt._actions[k] = rt._selfValue.Method(n)
	}
	if c._hasPreAction {
		rt._preAction = rt._interface.(goweb.ActionPreprocessor)
	}
	if c._hasPostAction {
		rt._postAction = rt._interface.(goweb.ActionPostprocessor)
	}
	return rt
}

func (c *schema) Init(ctl goweb.Controller, facs goweb.FactoryContainer) {
	c.Target = ctl
	rva := reflect.ValueOf(ctl)
	rtp := reflect.TypeOf(ctl)
	c._tValue = rva
	c._tType = rtp
	c._query = make(map[string]fieldType)
	c._actions = make(map[string]int)
	c._lft = ctl.Type()
	for rva.Kind() == reflect.Ptr {
		rva = rva.Elem()
	}
	c.initSubFields(c._tValue, []int{}, facs)
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
		c._actions[name] = i
	}
	if goweb.Debug {
		str := ""
		for k, _ := range c._actions {
			str += "[" + strings.ToUpper(k) + "] "
		}
		goweb.Log.Printf("init `%s` actions(%d) %s", rtp.Elem().Name(), len(c._actions), str)
	}
	if _, ok := c.Target.(goweb.ActionPreprocessor); ok {
		c._hasPreAction = true
	}
	if _, ok := c.Target.(goweb.ActionPostprocessor); ok {
		c._hasPostAction = true
	}
}

func (c *schema) initSubFields(value reflect.Value, index []int, facs goweb.FactoryContainer) {
	for reflect.Ptr == value.Kind() {
		value = value.Elem()
	}
	count := value.NumField()
	for i := 0; i < count; i++ {
		stfd := value.Type().Field(i) // struct field
		fdva := value.Field(i)        // field value
		tagName := stfd.Tag.Get(InjectTagName)
		if !fdva.CanSet() {
			continue
		}
		if tagName == "-" {
			continue
		}
		switch kind := stfd.Type.Kind(); kind {
		case reflect.Int, reflect.String, reflect.Float32, reflect.Bool, reflect.Slice:
			c._query[strings.ToLower(stfd.Name)] = fieldType{
				name: stfd.Type.PkgPath() + stfd.Type.Name(),
				id:   append(index, i),
			}
		case reflect.Ptr, reflect.Interface:
			if tagName == "" {
				if reflect.Ptr == kind {
					tagName = stfd.Type.Elem().PkgPath() + "/" + stfd.Type.Elem().Name()
				} else {
					tagName = stfd.Type.PkgPath() + "/" + stfd.Type.Name()
				}
			}
			switch facs.LookupType(tagName) {
			case goweb.LifeTypeStandalone:
				id := make([]int, len(index), len(index)+1)
				copy(id, index)
				c._standalone = append(c._standalone, fieldType{
					name: tagName,
					id:   append(id, i),
				})
			case goweb.LifeTypeStateful:
				id := make([]int, len(index), len(index)+1)
				copy(id, index)
				c._stateful = append(c._stateful, fieldType{
					name: tagName,
					id:   append(id, i),
				})
			case goweb.LifeTypeStateless:
				id := make([]int, len(index), len(index)+1)
				copy(id, index)
				c._stateless = append(c._stateless, fieldType{
					name: tagName,
					id:   append(id, i),
				})
			default:
				panic(goweb.NewWebError(http.StatusServiceUnavailable, "factory `%s` type can not be specified by alias `%s`", stfd.Type, tagName).ErrorAll())
			}
		case reflect.Struct:
			c.initSubFields(fdva, append(index, i), facs)
		}
	}
}

func isActionMethod(method *reflect.Method) bool {
	if !strings.HasPrefix(method.Name, ActionPrefix) {
		return false
	}
	if method.Type.NumIn() != 1 {
		if goweb.Debug {
			goweb.Log.Printf("`%s` is not a action because number in is %d != 0", method.Name, method.Type.NumIn()-1)
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
