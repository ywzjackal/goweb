package factory

import (
	"github.com/ywzjackal/goweb"
	"reflect"
)

const InjectTagName string = "inject"

type nodeType struct {
	nm string        // full struct name with package path
	dv reflect.Value // default value for this type
	id int           // field index
}

type schema struct {
	Target      goweb.Factory // Target must be set before register to container
	_tValue     reflect.Value // target(user defined controller struct) reflect.Value
	_tType      reflect.Type  // target(user defined controller struct) reflect.Type
	_standalone []nodeType    // factory which need be injected after first initialized
	_stateful   []nodeType    // factory which need be injected from session before called
	_stateless  []nodeType    // factory which need be injected always new before called
	_name       string        // package+struct full name
	_alias      string        // alias
}

func (s *schema) NewInjectAble(factory goweb.Factory) goweb.InjectAble {
	var nv reflect.Value
	if factory != nil {
		nv = reflect.ValueOf(factory)
	} else {
		nv = reflect.New(s._tType.Elem())
		factory = nv.Interface().(goweb.Factory)
	}
	able := &injectAble{
		paren: factory,
		fType: s._tType,
		fValu: nv,
		stata: make([]goweb.InjectNode, len(s._standalone), len(s._standalone)),
		statf: make([]goweb.InjectNode, len(s._stateful), len(s._stateful)),
		statl: make([]goweb.InjectNode, len(s._stateless), len(s._stateless)),
		fulnm: s._name,
	}
	for i, n := range s._standalone {
		able.stata[i].Name = n.nm
		able.stata[i].Value = nv.Elem().Field(n.id)
	}
	for i, n := range s._stateful {
		able.statf[i].Name = n.nm
		able.statf[i].Value = nv.Elem().Field(n.id)
	}
	for i, n := range s._stateless {
		able.statl[i].Name = n.nm
		able.statl[i].Value = nv.Elem().Field(n.id)
	}
	if init, ok := factory.(goweb.InitAble); ok {
		init.Init()
	}
	return able
}

func newSchema(factory goweb.Factory, alias string) (string, schema) {
	s := schema{
		Target:      factory,
		_tValue:     reflect.ValueOf(factory),
		_tType:      reflect.TypeOf(factory),
		_standalone: make([]nodeType, 0),
		_stateful:   make([]nodeType, 0),
		_stateless:  make([]nodeType, 0),
		_alias:      alias,
	}
	s._name = s._tType.Elem().PkgPath() + "/" + s._tType.Elem().Name()
	field_count := s._tValue.Elem().NumField()
	for i := 0; i < field_count; i++ {
		f := s._tType.Elem().Field(i)
		v := s._tValue.Elem().Field(i)
		tn := f.Tag.Get(InjectTagName) // target factory name with full package path
		if !v.CanSet() {
			continue
		}
		switch f.Type.Kind() {
		case reflect.Ptr:
			if tn == "" {
				tn = f.Type.Elem().PkgPath() + "/" + f.Type.Elem().Name()
			}
		case reflect.Interface:
			if tn == "" {
				goweb.Err.Printf("`%s.%s` no inject tag to determine wich factory will be used",
					f.Type.Name(), f.Name)
				continue
			}
		default:
			continue
		}
		tmpFac, ok := reflect.New(f.Type.Elem()).Interface().(goweb.Factory)
		if !ok {
			goweb.Err.Printf("`%s.%s` is not a type of factory!", s._tType.Elem().Name(), f.Name)
			continue
		}
		switch tmpFac.Type() {
		case goweb.LifeTypeStandalone:
			s._standalone = append(s._standalone, nodeType{
				nm: tn,
				id: i,
			})
		case goweb.LifeTypeStateful:
			s._stateful = append(s._standalone, nodeType{
				nm: tn,
				id: i,
			})
		case goweb.LifeTypeStateless:
			s._stateless = append(s._standalone, nodeType{
				nm: tn,
				id: i,
			})
		default:
			goweb.Err.Printf("`%s.Type()` return not a known value : %d", f.Type.Elem().Name(), tmpFac.Type())
			continue
		}
	}
	return s._name, s
}
