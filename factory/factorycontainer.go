package factory

import (
	"reflect"

	"github.com/ywzjackal/goweb"
)

type factoryContainer struct {
	goweb.FactoryContainer
	factorys map[reflect.Type]*factory
}

func NewFactoryContainer() goweb.FactoryContainer {
	return &factoryContainer{
		factorys: make(map[reflect.Type]*factory),
	}
}

var emptyValue = reflect.ValueOf(0)

func (f *factoryContainer) Register(faci goweb.Factory) goweb.WebError {
	var (
		t                  = reflect.TypeOf(faci)
		fac *factory       = nil
		err goweb.WebError = nil
		ok  bool           = false
	)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	fac, ok = f.factorys[t]
	if ok {
		return goweb.NewWebError(500, "Regist factory `%s` duplicate!", t.String())
	}
	if fac, err = f.initFactory(faci); err != nil {
		return err.Append(500, "Fail to Register factory `%s`!", t.String())
	}
	f.factorys[t] = fac
	return nil
}

func (f *factoryContainer) Lookup(rt reflect.Type, ctx goweb.Context) (reflect.Value, goweb.WebError) {
	var (
		err    goweb.WebError = nil // error
		target reflect.Value        // target which we are looking for
	)
	for rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
finding:
	fac, isexist := f.factorys[rt]
	if isexist {
		goto found
	} else {
		for _typ, _fac := range f.factorys {
			if _typ.AssignableTo(rt) {
				f.factorys[rt] = _fac
				fac = _fac
				goto found
			}
		}
	}
	// create new one!
	goweb.Log.Printf("Auto Register Factory `%s`", rt)
	target = reflect.New(rt)
	if err = f.Register(target.Interface()); err != nil {
		goweb.Err.Printf("Auto Register Factory `%s` Fail!", rt)
		return target, err.Append(500, "Auto Create Factory:%s Fail!", rt)
	}
	goto finding
found:
	switch fac._type {
	case goweb.LifeTypeStandalone:
	case goweb.LifeTypeStateful:
		if ctx == nil {
			return target, goweb.NewWebError(500, "Lookup Stateful Factory `%s` on non goweb.Context condition!", fac._selfValue.Type())
		}
		mem := ctx.Session().MemMap()
		itfs, ok := mem["__fac_"+rt.Name()]
		if !ok {
			target = reflect.New(fac._selfValue.Type())
			faci := target.Interface().(goweb.Factory)
			if fac, err = f.initFactory(faci); err != nil {
				return target, err.Append(500, "create stateful factory `%s` fail!", rt.Name())
			}
			mem["__fac_"+rt.Name()] = fac
		} else {
			fac, ok = itfs.(*factory)
			if !ok {
				return target, goweb.NewWebError(500, "can not restore stateful factory `%s` from session!", rt)
			}
		}
	case goweb.LifeTypeStateless:
		target = reflect.New(reflect.TypeOf(fac._selfValue.Type()).Elem())
		if fac, err = f.initFactory(target.Interface().(goweb.Factory)); err != nil {
			return target, err.Append(500, "create stateless factory `%s` fail!", rt)
		}
	default:
	}
	if err = resolveInjections(f, ctx, fac._stateful); err != nil {
		return target, err.Append(500, "Fail to resolve injection for factory `%s`", fac._selfValue.Type())
	}
	if err = resolveInjections(f, ctx, fac._stateless); err != nil {
		return target, err.Append(500, "Fail to resolve injection for factory `%s`", fac._selfValue.Type())
	}
	return fac._selfValue.Addr(), err
}

func (f *factoryContainer) initFactory(faci goweb.Factory) (*factory, goweb.WebError) {
	var (
		t   reflect.Type  = reflect.TypeOf(faci)
		v   reflect.Value = reflect.ValueOf(faci)
		fac *factory      = &factory{
		//			_selfValue: v,
		}
		facVal reflect.Value = reflect.ValueOf(fac)
	)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	fac._intFunc = v.Addr().MethodByName("Init")
	fac._selfValue = v
	for i := 0; i < v.NumField(); i++ {
		stfd := v.Type().Field(i) // struct field
		fdva := v.Field(i)        // field value
		if !fdva.CanSet() {
			continue
		}
		if stfd.Anonymous {
			if fdva.Type().Kind() == reflect.Interface {
				switch fdva.Type().Name() {
				case "FactoryStandalone":
					fac._type = (goweb.LifeTypeStandalone)
				case "FactoryStateful":
					fac._type = (goweb.LifeTypeStateful)
				case "FactoryStateless":
					fac._type = (goweb.LifeTypeStateless)
				default:
				}
				switch {
				case facVal.Type().AssignableTo(fdva.Type()):
					fdva.Set(facVal)
					goweb.Log.Printf("Assign `%s` to `%s` of `%s`:`%d`", facVal.Type(), fdva.Type(), v.Type(), i)
				default:
				}
			}
			continue
		}
		switch stfd.Type.Kind() {
		case reflect.Interface, reflect.Ptr:
			if isTypeLookupAble(stfd.Type) != nil {
				break
			}
			switch factoryType(stfd.Type) {
			case goweb.LifeTypeStandalone:
				// look up standalone factory when initialize
				_v, err := f.Lookup(stfd.Type, nil)
				if err != nil {
					return nil, err.Append(500, "Fail to initialize `%s`'s field `%s`", v.Type(), stfd.Type)
				}
				fdva.Set(_v)
				fac._standalone = append(fac._stateful, injectNode{
					id: i,
					tp: stfd.Type,
					va: &fdva,
				})
			case goweb.LifeTypeStateful:
				fac._stateful = append(fac._stateful, injectNode{
					id: i,
					tp: stfd.Type,
					va: &fdva,
				})
			case goweb.LifeTypeStateless:
				fac._stateless = append(fac._stateful, injectNode{
					id: i,
					tp: stfd.Type,
					va: &fdva,
				})
			default:
				return nil, goweb.NewWebError(500, "Factory `%s` type not specified, need specified by FactoryStandalone/FactoryStateless/FactoryStateful", stfd.Type)
			}
		}
	}
	switch fac._type {
	case goweb.LifeTypeStateless:
		if len(fac._stateful) != 0 {
			return nil, goweb.NewWebError(500, "Stateless `%s` can not inject stateful field `%s`", fac._selfValue.Type(), fac._stateful[0].tp)
		}
	case goweb.LifeTypeStandalone:
		if len(fac._stateful) != 0 {
			return nil, goweb.NewWebError(500, "Standalone `%s` can not inject stateful field `%s`", fac._selfValue.Type(), fac._stateful[0].tp)
		}
		fac.Init()
	case goweb.LifeTypeStateful:
	case goweb.LifeTypeError:
		return nil, goweb.NewWebError(500, "Factory need extend from one of interface FactoryStandalone/FactoryStateful/FactoryStateless")
	}

	return fac, nil
}

func resolveInjections(factorys goweb.FactoryContainer, ctx goweb.Context, nodes []injectNode) goweb.WebError {
	for _, node := range nodes {
		v, err := factorys.Lookup(node.tp, ctx)
		if err != nil {
			return err.Append(500, "Fail to inject `%s`", node.tp)
		}
		if !v.IsValid() {
			return goweb.NewWebError(500, "inject invalid value to `%s`", node.va)
		}
		if v.Kind() != reflect.Ptr {
			return goweb.NewWebError(500, "inject invalid type of %s, need Ptr", v.Kind())
		}
		node.va.Set(v)
	}
	return nil
}
