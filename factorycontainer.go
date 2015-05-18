package goweb

import "reflect"

// FactoryContainer is the container interface of factorys
type FactoryContainer interface {
	// Init after create new instanse.
	// notice:
	// 		fc := &FactoryContainerStruct{}
	// 		fc.Init() //！！Must be called after created！！
	// 初始化，当工厂容器被创建后，必须马上使用本函数初始化
	Init() WebError
	// RegisterFactory will register a new factory for Lookup later.
	// 注册工厂，以供以后查询（Lookup）使用
	Register(Factory) WebError
	// Lookup factory by type from this container or depends and return it.
	// Lookup also enject Ptr or Interface fields which is Exported and
	// Setable for the factory be looking up.
	// if something wrong happend, the error will be set.
	// for Standalone Factory, Lookup will return the global factory which
	// can be assignable to the type of rt.
	// for Stateful Factory, Lookup will find in session, if it doesn't exist
	// in session, Lookup will create new one and add to session for later.
	// for Stateless Factory, Lookup will allways create new instance by rt,
	// and never store in container or session.
	// 根据工厂的类型（reflect.type）从容器或depends中查找与之相配的值并返回。
	// 如果工厂中有导出并且可以被赋值的指针（Prt）或接口（Interface）域（Field），
	// Lookup 自动注册这些域。
	// 对于独立工厂，Lookup将从全局唯一的实例返回，
	// 对于有状态工厂，Lookup将先从会话中查找，如果没有将创建一个新的工厂实例并添加到会话
	// 中，以供以后的使用。
	// 对于无状态工厂，Lookup始终返回一个刚被新创建的实例。
	Lookup(rt reflect.Type, context Context) (reflect.Value, WebError)
}

type factoryContainer struct {
	FactoryContainer
	factorys map[reflect.Type]*factory
}

func (f *factoryContainer) Init() WebError {
	Log.Printf("INIT FC...")
	f.factorys = make(map[reflect.Type]*factory)
	return nil
}

var emptyValue = reflect.ValueOf(0)

func (f *factoryContainer) Register(faci Factory) WebError {
	var (
		t            = reflect.TypeOf(faci)
		fac *factory = nil
		err WebError = nil
		ok  bool     = false
	)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	fac, ok = f.factorys[t]
	if ok {
		return NewWebError(500, "Regist factory `%s` duplicate!", t.String())
	}
	if fac, err = f.initFactory(faci); err != nil {
		return err.Append(500, "Fail to Register factory `%s`!", t.String())
	}
	f.factorys[t] = fac
	return nil
}

func (f *factoryContainer) Lookup(rt reflect.Type, ctx Context) (reflect.Value, WebError) {
	var (
		err    WebError      = nil // error
		target reflect.Value       // target which we are looking for
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
	Log.Printf("Auto Register Factory `%s`", rt)
	target = reflect.New(rt)
	if err = f.Register(target.Interface()); err != nil {
		Err.Printf("Auto Register Factory `%s` Fail!", rt)
		return target, err.Append(500, "Auto Create Factory:%s Fail!", rt)
	}
	goto finding
found:
	switch fac._type {
	case LifeTypeStandalone:
	case LifeTypeStateful:
		if ctx == nil {
			return target, NewWebError(500, "Lookup Stateful Factory `%s` on non Context condition!", fac._selfValue.Type())
		}
		mem := ctx.Session().MemMap()
		itfs, ok := mem["__fac_"+rt.Name()]
		if !ok {
			target = reflect.New(fac._selfValue.Type())
			faci := target.Interface().(Factory)
			if fac, err = f.initFactory(faci); err != nil {
				return target, err.Append(500, "create stateful factory `%s` fail!", rt.Name())
			}
			mem["__fac_"+rt.Name()] = fac
		} else {
			fac, ok = itfs.(*factory)
			if !ok {
				return target, NewWebError(500, "can not restore stateful factory `%s` from session!", rt)
			}
		}
	case LifeTypeStateless:
		target = reflect.New(reflect.TypeOf(fac._selfValue.Type()).Elem())
		if fac, err = f.initFactory(target.Interface().(Factory)); err != nil {
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

func (f *factoryContainer) initFactory(faci Factory) (*factory, WebError) {
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
					fac._type = (LifeTypeStandalone)
				case "FactoryStateful":
					fac._type = (LifeTypeStateful)
				case "FactoryStateless":
					fac._type = (LifeTypeStateless)
				default:
				}
				switch {
				case facVal.Type().AssignableTo(fdva.Type()):
					fdva.Set(facVal)
					Log.Printf("Assign `%s` to `%s` of `%s`:`%d`", facVal.Type(), fdva.Type(), v.Type(), i)
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
			case LifeTypeStandalone:
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
			case LifeTypeStateful:
				fac._stateful = append(fac._stateful, injectNode{
					id: i,
					tp: stfd.Type,
					va: &fdva,
				})
			case LifeTypeStateless:
				fac._stateless = append(fac._stateful, injectNode{
					id: i,
					tp: stfd.Type,
					va: &fdva,
				})
			default:
				return nil, NewWebError(500, "Factory `%s` type not specified, need specified by FactoryStandalone/FactoryStateless/FactoryStateful", stfd.Type)
			}
		}
	}
	switch fac._type {
	case LifeTypeStateless:
		if len(fac._stateful) != 0 {
			return nil, NewWebError(500, "Stateless `%s` can not inject stateful field `%s`", fac._selfValue.Type(), fac._stateful[0].tp)
		}
	case LifeTypeStandalone:
		if len(fac._stateful) != 0 {
			return nil, NewWebError(500, "Standalone `%s` can not inject stateful field `%s`", fac._selfValue.Type(), fac._stateful[0].tp)
		}
		fac.Init()
	case LifeTypeStateful:
	case LifeTypeError:
		return nil, NewWebError(500, "Factory need extend from one of interface FactoryStandalone/FactoryStateful/FactoryStateless")
	}

	return fac, nil
}
