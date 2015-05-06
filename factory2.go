package goweb

import "reflect"

const (
	FactoryAutoInitFuncName = "init_auto"
)

func (f *factoryContainer) RegisterFactory(fi interface{}) WebError {
	var (
		t            = reflect.TypeOf(fi) // reflect.type
		err WebError = nil
	)
	if err = isTypeRegisterAble(t); err != nil {
		return err.Append(500, "RegisterFactory must be a Pointer of struct:(*struct)!")
	}
	for _, _w := range f.factorys {
		if _w.value == fi {
			return NewWebError(1, "Resigter `%s` duclicate!", fi)
		}
	}
	wrap := newFactoryWrap(fi)
	if wrap == nil {
		return NewWebError(1, "RegisterFactory must be a Pointer of Factory! got %s", t)
	}
	err = f.factoryInitilazion(&wrap.rv, wrap, nil)
	if err != nil {
		return err.Append(1, "Fail to RegisterFactory `%s`", wrap.rt)
	}
	f.factorys = append(f.factorys, wrap)
	return nil
}

func (f *factoryContainer) lookupWithoutAutoRegister(rt reflect.Type, ctx Context) (reflect.Value, WebError) {
	var (
		tw     *factoryWrap  = nil   // tw:targetWrap
		iaec   bool          = false // is auto init func exist in container
		err    WebError      = nil   // error
		target reflect.Value         // target which we are looking for
	)
	if err = isTypeLookupAble(rt); err != nil {
		return target, err.Append(500, "Fail to lookup '%s'", rt)
	}
	for _, _w := range f.factorys {
		if _w.rt.AssignableTo(rt) {
			//		if _w.rt == rt {
			tw = _w
			break
		}
	}
	// check if factory which we are looking for is not exist in container.
	if !iaec {
		return target, NewWebError(500, "Not found factory:%s", rt)
	}
	switch tw.state {
	case FactoryTypeStateful:
		// inject from client session, if not found register new to session
	case FactoryTypeStandalone:
		// inject from factory container
		target = tw.rv
	case FactoryTypeStateless:
		// crate a new factory for target every times
		target = reflect.New(tw.rt)
		err = f.factoryInitilazion(&target, tw, ctx)
		if err != nil {
			return target, err.Append(1, "Fail to initilazion `%s`", target.Type())
		}
	}
	err = f.factoryInitilazion(&target, tw, ctx)
	return target, err
}

func (f *factoryContainer) Lookup(rt reflect.Type, ctx Context) (reflect.Value, WebError) {
	var (
		tw     *factoryWrap  = nil   // tw:targetWrap
		iaec   bool          = false // is auto init func exist in container
		err    WebError      = nil   // error
		target reflect.Value         // target which we are looking for
	)
	if err = isTypeLookupAble(rt); err != nil {
		return target, err.Append(500, "Fail to lookup '%s'", rt)
	}
	for _, _w := range f.factorys {
		if _w.rt.AssignableTo(rt) {
			//		if _w.rt == rt {
			tw = _w
			break
		}
	}
	// check if factory which we are looking for is not exist in container.
	if !iaec {
		t := rt
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		Log.Printf("`%s` doesn't exist in factory container, auto register and initialization", t.Name())
		// auto register this factory
		err = f.RegisterFactory(reflect.New(t).Interface())
		if err != nil {
			return target, err.Append(1, "register `%s` fail!", t.Name())
		}
		target, err = f.lookupWithoutAutoRegister(rt, ctx)
		if err != nil {
			return target, err.Append(500, "Fail to auto register factory `%s`", target)
		}
	}
	switch tw.state {
	case FactoryTypeStateful:
		// inject from client session, if not found register new to session
	case FactoryTypeStandalone:
		// inject from factory container
		target = tw.rv
	case FactoryTypeStateless:
		// crate a new factory for target every times
		target = reflect.New(tw.rt)
		err = f.factoryInitilazion(&target, tw, ctx)
		if err != nil {
			return target, err.Append(1, "Fail to initilazion `%s`", target.Type())
		}
	}
	err = f.factoryInitilazion(&target, tw, ctx)
	return target, err
}

///////////// tools //////////////

func newFactoryWrap(fi interface{}) *factoryWrap {
	var (
		iav       []reflect.Value                           //factory init_auto paramters reflect.type
		iat       []reflect.Type                            //auto init function method pointer
		aifm      *reflect.Method                           // auto init func method pointer
		t         = reflect.TypeOf(fi)                      // reflect.type
		v         = reflect.ValueOf(fi)                     // reflect.value
		s         = factoryType(t)                          // FactoryType
		aif, haif = t.MethodByName(FactoryAutoInitFuncName) // auto int func,has auto int func
		argslen   = 0                                       // auto init function parameters in count
	)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		v = v.Elem()
	}
	if s == FactoryTypeError {
		return nil
	}
	wrap := &factoryWrap{
		rt:           t,
		rv:           v,
		value:        fi,
		state:        s,
		aifm:         aifm,
		initArgs:     iav,
		initArgsType: iat,
		injectsSa:    []reflect.Value{},
		injectsSf:    []reflect.Value{},
		injectsSl:    []reflect.Value{},
	}
	if haif {
		aifm = &aif
		argslen = aifm.Func.Type().NumIn()
		iav = make([]reflect.Value, argslen)
		iat = make([]reflect.Type, argslen)
		for i := 0; i < argslen; i++ {
			iat[i] = aifm.Func.Type().In(i)
		}
	}
	for i := 0; i < v.NumField(); i++ {
		_v := v.Field(i)
		_t := t.Field(i)
		if _t.Anonymous {
			continue
		}
		if !_v.CanSet() {
			continue
		}
		if _v.Kind() == reflect.Ptr || _v.Kind() == reflect.Interface {
			switch factoryType(_v.Type()) {
			case FactoryTypeStandalone:
				wrap.injectsSa = append(wrap.injectsSa, _v)
			case FactoryTypeStateful:
				wrap.injectsSf = append(wrap.injectsSf, _v)
			case FactoryTypeStateless:
				wrap.injectsSl = append(wrap.injectsSl, _v)
			}
		}
	}
	return wrap
}

func (f *factoryContainer) factoryInitilazion(fv *reflect.Value, fw *factoryWrap, context Context) WebError {
	var (
		airt []reflect.Value // auto init result
		err  WebError
		b    bool
	)
	// call init_auto func
	if fw.aifm != nil {
		// need auto initialization
		fw.initArgs[0] = fw.rv
		for i, t := range fw.initArgsType[1:] {
			fw.initArgs[i+1], err = f.Lookup(t, context)
			if err != nil {
				return err.Append(1, "Fail to lookup `%s`", t)
			}
		}
		airt = fw.aifm.Func.Call(fw.initArgs)
		if len(airt) == 1 {
			err, b = airt[0].Interface().(WebError)
			if b && err != nil {
				return err.Append(1, "Fail to Call init func,factory:'%s'", fw.rt)
			}
		}
	}
	for _, v := range fw.injectsSa {
		v, err = f.Lookup(v.Type(), context)
		if err != nil {
			return err.Append(1, "Fail to inject `%s`'s `%s`", fw.rt, v.Type())
		}
	}
	//
	return nil
}

func factoryType(t reflect.Type) FactoryType {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return FactoryTypeError
	}
	if _, b := t.FieldByName("FactoryStateful"); b {
		return (FactoryTypeStateful)
	} else if _, b := t.FieldByName("FactoryTypeStandalone"); b {
		return (FactoryTypeStandalone)
	} else {
		return (FactoryTypeStateless)
	}
}
