package goweb

import (
	"fmt"
	"reflect"
)

// FactoryStandalone is the factory that only one instance in system，all the
// Enjection share on the same factory instance. so be carful for thread-safe
// 独立状态工厂接口，仅有一个实例存在与系统中，所有注入共享一个实例，所以必要时应该考虑线程
// 安全。
type FactoryStandalone struct{}

// FactoryStateful is stateful for user in session. each session has the one
// and only instance for itself, no shared, and will auto destroy when
// session timeout or be destroyed.
// 有状态工厂接口，面向用户的有状态工厂，每个会话（SESSION）包含一个与众不同的实例，不共
// 享，当会话（SESSION）超时或摧毁时包含的有状态的工厂也将被自动摧毁
type FactoryStateful struct{}

// FactoryStateless is stateless for user in session. Enjection will allways
// create a new factory instance for using.
// 无状态工厂接口，面向用户无状态，每次注入（Enject）将创建一个新的实例以供调用。
type FactoryStateless struct {
	context Context
}

func (f FactoryStateless) Context() Context {
	return f.context
}

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
	RegisterFactory(interface{}) WebError
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

type factoryWrap struct {
	value        interface{}     // store interface
	rt           reflect.Type    // store reflect.type, not pointer or interface
	rv           reflect.Value   // store reflect.value,not pointer or interface
	state        FactoryType     // FactoryTypeStandalone/FactoryTypeStateful/FactoryTypeStateless
	initArgs     []reflect.Value // store factory init_auto paramters reflect.value
	initArgsType []reflect.Type  // store factory init_auto paramters reflect.type
	aifm         *reflect.Method // store auto init function method pointer
	injectsSa    []reflect.Value // fields whose need be inject @Standalone
	injectsSl    []reflect.Value // fields whose need be inject @Standless
	injectsSf    []reflect.Value // fields whose need be inject @Standful
}

type factoryContainer struct {
	FactoryContainer
	factorys []*factoryWrap
}

func (f *factoryContainer) Init() WebError {
	Log.Printf("INIT FC...")
	f.factorys = make([]*factoryWrap, 0)
	return nil
}

func (f *factoryContainer) RegisterFactory3(fi interface{}) {
	t := reflect.TypeOf(fi)
	v := reflect.ValueOf(fi)
	if t.Kind() != reflect.Ptr {
		panic(fmt.Errorf("RegisterFactory must be a Pointer of Factory! got %s",
			t))
		return
	}
	if v.IsNil() {
		panic(fmt.Errorf("RegisterFactory must not nil!"))
		return
	}
	if !v.IsValid() {
		panic(fmt.Errorf("RegisterFactory must valid!"))
		return
	}
	fname := t.Elem().Name()
	wrap := &factoryWrap{
		rt:    t,
		rv:    v,
		value: fi,
	}
	f.factorys = append(f.factorys, wrap)
	Log.Printf("Register Factory `%s`", fname)
}

var emptyValue = reflect.ValueOf(0)

func (f *factoryContainer) Lookup3(
	rt reflect.Type, context Context) (reflect.Value, WebError) {
	if rt.Kind() == reflect.Ptr || rt.Kind() == reflect.Interface {
		v, _, e := f.lookup(nil, rt, context)
		return v, e
	}
	return emptyValue,
		NewWebError(1, fmt.Sprintf("Lookup %s use %s, except Pointer or Interface!", rt, rt.Kind()), nil)
}

func (f *factoryContainer) lookup(loopTree []string, rt reflect.Type,
	depends ...interface{}) (reflect.Value, []string, WebError) {
	var (
		value         reflect.Value                         // target reflect.value
		valueType     reflect.Type                          // target reflect.type
		rtname        = rt.Name()                           // type name
		dependstValue = make([]reflect.Value, len(depends)) // depends reflect.value
	)
	// if rt is a pointer, rt.elem.name is the real name
	// else(interface) use rt.name
	if rt.Kind() == reflect.Ptr {
		rtname = rt.Elem().Name()
	}
	// inject loop initial
	if loopTree == nil {
		loopTree = []string{}
	}
	// set max to 100 deep of loop tree
	if len(loopTree) > 100 {
		return emptyValue, loopTree, NewWebError(1,
			fmt.Sprintf("Enjection may Deadloop!:%s", loopTree), nil)
	}
	// check deadloop
	for _, pre := range loopTree {
		if pre == rtname {
			return emptyValue, loopTree, NewWebError(1,
				fmt.Sprintf("Enjection may Deadloop!:%s", loopTree), nil)
		}
	}
	// append current value to looptree
	loopTree = append(loopTree, rtname)
	// Decompression reflect.value from depends to dependstValue
	// and try inject from depends before factory container
	for i, depend := range depends {
		dependstValue[i] = reflect.ValueOf(depend)
		if dependstValue[i].Type().AssignableTo(rt) {
			return dependstValue[i], loopTree, nil
		}
	}
	// if not found in depends, now try from factory container
	for _, v := range f.factorys {
		// try if container's element can assignable to target
		if v.rt.AssignableTo(rt) {
			value = v.rv
			valueType = v.rt
			goto found
		}
		// if element not assignable to target,
		// try element's anonymous fields (derivative)
		for i := 0; i < v.rt.Elem().NumField(); i++ {
			ft := v.rt.Elem().Field(i)
			fv := v.rv.Elem().Field(i)
			if ft.Anonymous && ft.Type.AssignableTo(rt) {
				value = fv
				valueType = ft.Type
				goto found
			}
		}
	}
	// not found target from depends and factory container, now we have to
	// create a new one
	Err.Printf("Not found `%s` in Container! Auto register without initial!", rt)
	value = reflect.New(rt.Elem())
	valueType = rt
	f.RegisterFactory(value.Interface())

found:
	if !value.Elem().IsValid() {
		Err.Printf("%s is invalid!", valueType.Elem().Name())
		value = reflect.New(valueType.Elem())
	}
	// now value and valueType of target is ready, try initailization it
	for i := 0; i < valueType.Elem().NumField(); i++ {
		fv := value.Elem().Field(i)
		ft := valueType.Elem().Field(i)
		// igno anonymous fields
		if ft.Anonymous {
			continue
		}
		// igno private fields
		if !fv.CanSet() {
			continue
		}
		// igno fields if they are not Pointer or Interface
		if fv.Type().Kind() != reflect.Ptr &&
			fv.Type().Kind() != reflect.Interface {
			continue
		}
		// if they already not nil, why we reinitialization it?
		if !fv.IsNil() {
			continue
		}
		// try inject from depends first
		for j, depend := range dependstValue {
			if depend.Type().AssignableTo(fv.Type()) {
				fv.Set(dependstValue[j])
				goto foundfromdepends
			}
		}
		{
			// and if not found, try inject from factory container
			factory, loopTree, err := f.lookup(loopTree, fv.Type())
			if err != nil {
				return emptyValue, loopTree,
					NewWebError(1, fmt.Sprintf("Can not Enject `%s.%s`,'%s'",
						valueType, fv, err.Error()), nil)
			}
			fv.Set(factory)
		}
	foundfromdepends:
		continue
	}
	return value, loopTree, nil
}

const (
	FactoryTypeStateless = iota
	FactoryTypeStandalone
	FactoryTypeStateful
	FactoryTypeError
)

type FactoryType int
