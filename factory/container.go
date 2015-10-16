package factory

import (
	"github.com/ywzjackal/goweb"
	"net/http"
	"reflect"
)

type factoryContainer struct {
	goweb.FactoryContainer
	scmas map[string]schema // all registered factory map with its full name
	stand map[string]goweb.InjectAble
}

func NewFactoryContainer() goweb.FactoryContainer {
	return &factoryContainer{
		scmas: make(map[string]schema),
		stand: make(map[string]goweb.InjectAble),
	}
}

func (f *factoryContainer) RegisterWithReflectType(typ reflect.Type, alias string) {
	able, ok := reflect.New(typ.Elem()).Interface().(goweb.InjectAble)
	if !ok {
		panic("auto register fail: " + typ.Elem().Name())
	}
	f.Register(able, alias)
}

func (f *factoryContainer) Register(factory goweb.Factory, alias string) {
	name, sch := newSchema(factory, alias, f)
	if goweb.Debug {
		goweb.Log.Printf("register `%s:%s` type:%s", name, alias, factory.Type())
	}
	//
	if _, ok := f.scmas[name]; ok {
		panic("dumplaction register factory `" + name + "`")
	}
	f.scmas[name] = sch
	if factory.Type() == goweb.LifeTypeStandalone {
		f.stand[name] = sch.NewInjectAble(factory)
		factory, err := f.Lookup(name, nil)
		if err != nil {
			panic(err)
		}
		if init, ok := factory.(goweb.InitAble); ok {
			init.Init()
		} else {
			goweb.Log.Printf("standalone factory `%s` is not goweb.InitAble", name)
		}
	}
	//
	if alias != "" && alias != name {
		if _, ok := f.scmas[alias]; ok {
			panic("dumplaction register factory `" + alias + "`")
		}
		f.scmas[alias] = sch
		if factory.Type() == goweb.LifeTypeStandalone {
			f.stand[alias] = f.stand[name]
		}
	}
}

func (f *factoryContainer) LookupStandalone(name string) (goweb.InjectAble, goweb.WebError) {
	able, ok := f.stand[name]
	if !ok {
		s := ""
		if goweb.Debug {
			s += "found in:["
			for n, a := range f.stand {
				s += n + ":" + a.ReflectType().Elem().Name() + ","
			}
			s += "]"
		}
		return nil, goweb.NewWebError(http.StatusServiceUnavailable, "factory(\""+name+"\") not found,"+s)
	}
	if err := f.injectStandalone(able); err != nil {
		return nil, err.Append("fail to inject standalone field")
	}
	return able, nil
}

func (f *factoryContainer) LookupStateless(name string) (goweb.InjectAble, goweb.WebError) {
	sch, ok := f.scmas[name]
	if !ok {
		return nil, goweb.NewWebError(http.StatusServiceUnavailable, "schema not found")
	}
	able := sch.NewInjectAble(nil)
	if err := f.injectStandalone(able); err != nil {
		return nil, err.Append("fail to inject standalone field")
	}
	if err := f.injectStateless(able); err != nil {
		return nil, err.Append("fail to inject stateless field")
	}
	return able, nil
}

func (f *factoryContainer) LookupStateful(name string, state goweb.InjectGetterSetter) (goweb.InjectAble, goweb.WebError) {
	if able := state.Get(name); able != nil {
		return able, nil
	}
	able, err := f.LookupStateless(name)
	if err != nil {
		return nil, err.Append("first initialization failed")
	}
	state.Set(name, able)

	if err := f.injectStandalone(able); err != nil {
		return nil, err.Append("fail to inject standalone field")
	}
	if err := f.injectStateless(able); err != nil {
		return nil, err.Append("fail to inject stateless field")
	}
	if err := f.injectStateful(able, state); err != nil {
		return nil, err.Append("fail to inject stateful field")
	}
	return able, nil
}

func (f *factoryContainer) Lookup(alias string, state goweb.InjectGetterSetter) (goweb.Factory, goweb.WebError) {
	sch, ok := f.scmas[alias]
	if !ok {
		return nil, goweb.NewWebError(http.StatusServiceUnavailable, "schema not found")
	}
	var able goweb.InjectAble
	var err goweb.WebError
	switch sch.Target.Type() {
	case goweb.LifeTypeStandalone:
		able, err = f.LookupStandalone(alias)
		if err != nil {
			return nil, err
		}
	case goweb.LifeTypeStateless:
		able, err = f.LookupStateless(alias)
		if err != nil {
			return nil, err
		}
	case goweb.LifeTypeStateful:
		if state == nil {
			return nil, goweb.NewWebError(http.StatusInternalServerError, "can not lookup stateful factory without stateful container!")
		}
		able, err = f.LookupStateful(alias, state)
		if err != nil {
			return nil, err
		}
	default:
		return nil, goweb.NewWebError(http.StatusInternalServerError, "invalid factory type!")
	}
	fac, ok := able.ReflectValue().Interface().(goweb.Factory)
	if !ok {
		return nil, goweb.NewWebError(http.StatusInternalServerError, "can not convert '%s' to goweb.Factory", able.ReflectValue().Interface())
	}
	return fac, nil
}

func (f *factoryContainer) injectStandalone(able goweb.InjectAble) goweb.WebError {
	nods := able.FieldsStandalone()
	for _, n := range nods {
		nv, err := f.LookupStandalone(n.Name)
		if err != nil {
			return err.Append("lookup (%s:%s) fail", n.Name, n.Value.Type().Elem().PkgPath()+"/"+n.Value.Type().Elem().Name())
		}
		n.Value.Set(nv.ReflectValue())
	}
	return nil
}

func (f *factoryContainer) injectStateless(able goweb.InjectAble) goweb.WebError {
	nods := able.FieldsStateless()
	for _, n := range nods {
		nv, err := f.LookupStateless(n.Name)
		if err != nil {
			return err.Append("lookup (%s) fail", n.Value.Type().Name())
		}
		n.Value.Set(nv.ReflectValue())
	}
	return nil
}

func (f *factoryContainer) injectStateful(able goweb.InjectAble, state goweb.InjectGetterSetter) goweb.WebError {
	nods := able.FieldsStateful()
	for _, n := range nods {
		nv, err := f.LookupStateful(n.Name, state)
		if err != nil {
			return err.Append("lookup (%s) fail", n.Value.Type().Name())
		}
		n.Value.Set(nv.ReflectValue())
	}
	return nil
}
