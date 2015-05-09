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
	Registe(Factory) WebError
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
		t = reflect.TypeOf(faci)
	)
	_, duplicate := f.factorys[t]
	if duplicate {
		return NewWebError(500, "Regist factory `%s` duplicate!", t)
	}
	if err := f.initFactory(faci); err != nil {
		return err.Append(500, "Fail to Register factory `%s`!", t)
	}
	return nil
}

func (f *factoryContainer) Lookup(rt reflect.Type, ctx Context) (reflect.Value, WebError) {
	var (
		err    WebError      = nil // error
		target reflect.Value       // target which we are looking for
	)
	fac, isexist := f.factorys[rt]
	if !isexist {
		for _tp, _fac := range f.factorys {
			if _tp.AssignableTo(rt) {
				f.factorys[rt] = _fac
				fac = _fac
				goto found
			}
		}
	}
	return target, NewWebError(500, "Not found Factory:%s", rt)
found:
	switch fac._type {
	case LifeTypeStandalone:
		return fac._selfValue, nil
	case LifeTypeStateful:
		mem := ctx.Session().MemMap()
		_target, isexist := mem["__fac_"+rt.Name()]
		if !isexist {
			target = reflect.New(reflect.TypeOf(fac._selfValue.Type()).Elem())
			if err := f.initFactory(target.Interface().(Factory)); err != nil {
				return target, err.Append(500, "create stateful factory `%s` fail!", rt)
			}
			mem["__fac_"+rt.Name()] = target
			return target, nil
		}
		target, ok := _target.(reflect.Value)
		if !ok {
			return target, NewWebError(500, "can not restore stateful factory `%s` from session!", rt)
		}
		return target, nil
	case LifeTypeStateless:
		target = reflect.New(reflect.TypeOf(fac._selfValue.Type()).Elem())
		if err := f.initFactory(target.Interface().(Factory)); err != nil {
			return target, err.Append(500, "create stateless factory `%s` fail!", rt)
		} else {
			return target, nil
		}
	default:
	}
	return fac._selfValue, err
}

func (f *factoryContainer) lookupStandalone(rt reflect.Type) (reflect.Value, WebError) {
	var (
		err    WebError      = nil // error
		target reflect.Value       // target which we are looking for
	)

	return target, err
}

func (f *factoryContainer) initFactory(faci Factory) WebError {
	var (
		//		t   reflect.Type  = reflect.TypeOf(faci)
		v   reflect.Value = reflect.ValueOf(faci)
		fac *factory      = &factory{
			_selfValue: v,
		}
		facVal reflect.Value = reflect.ValueOf(fac)
	)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
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
				default:
					Log.Print(facVal.Type(), fdva.Type())
					return NewWebError(500, "interface %s of controller %s can not be assignable!", fdva.Type(), facVal.Type())
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
				_v, err := f.lookupStandalone(stfd.Type)
				if err != nil {
					return err.Append(500, "Fail to initialize `%s`'s field `%s`", v.Type(), stfd.Type)
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
				return NewWebError(500, "Factory `%s` type not specified, need specified by FactoryStandalone/FactoryStateless/FactoryStateful")
			}
		}
	}
	return nil
}
