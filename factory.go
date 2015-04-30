package goweb

import (
	"fmt"
	"reflect"
)

// FactoryStandalone is the factory that only one instance in system，all the
// Enjection share on the same factory instance. so be carful for thread-safe
// 独立状态工厂接口，仅有一个实例存在与系统中，所有注入共享一个实例，所以必要时应该考虑线程
// 安全。
type FactoryStandalone interface{}

// FactoryStateful is stateful for user in session. each session has the one
// and only instance for itself, no shared, and will auto destroy when
// session timeout or be destroyed.
// 有状态工厂接口，面向用户的有状态工厂，每个会话（SESSION）包含一个与众不同的实例，不共
// 享，当会话（SESSION）超时或摧毁时包含的有状态的工厂也将被自动摧毁
type FactoryStateful interface{}

// FactoryStateless is stateless for user in session. Enjection will allways
// create a new factory instance for using.
// 无状态工厂接口，面向用户无状态，每次注入（Enject）将创建一个新的实例以供调用。
type FactoryStateless interface{}

// FactoryContainer is the container interface of factorys
type FactoryContainer interface {
	// Init after create new instanse.
	// notice:
	// 		fc := &FactoryContainerStruct{}
	// 		fc.Init() //！！Must be called after created！！
	// 初始化，当工厂容器被创建后，必须马上使用本函数初始化
	Init() error
	// RegisterFactory will register a new factory for Lookup later.
	// 注册工厂，以供以后查询（Lookup）使用
	RegisterFactory(interface{})
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
	Lookup(rt reflect.Type, depends ...interface{}) (reflect.Value, error)
}

type factoryWrap struct {
	value interface{}
	rt    reflect.Type
	rv    reflect.Value
}

type factoryContainer struct {
	FactoryContainer
	factorys []*factoryWrap
}

func (f *factoryContainer) Init() error {
	Log.Printf("INIT FC...")
	f.factorys = make([]*factoryWrap, 0)
	f.RegisterFactory(&Factory{})
	return nil
}

func (f *factoryContainer) RegisterFactory(fi interface{}) {
	t := reflect.TypeOf(fi)
	v := reflect.ValueOf(fi)
	_, ok := fi.(*Factory)
	if !ok {
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

func (f *factoryContainer) Lookup(
	rt reflect.Type, depends ...interface{}) (reflect.Value, error) {
	if rt.Kind() == reflect.Ptr || rt.Kind() == reflect.Interface {
		v, _, e := f.lookup(nil, rt, depends...)
		return v, e
	}
	return emptyValue,
		fmt.Errorf("Lookup %s use %s, except Pointer or Interface!", rt, rt.Kind())
}

func (f *factoryContainer) lookup(loopTree []string, rt reflect.Type,
	depends ...interface{}) (reflect.Value, []string, error) {
	if loopTree == nil {
		loopTree = []string{}
	}
	if len(loopTree) > 100 {
		e := fmt.Errorf("Enjection Deadloop!:%s", loopTree)
		Err.Printf("%s", e)
		return emptyValue, loopTree, e
	}
	for _, pre := range loopTree {
		if pre == rt.Elem().Name() {
			e := fmt.Errorf("Enjection Deadloop!:%s", loopTree)
			Err.Printf("%s", e)
			return emptyValue, loopTree, e
		}
	}
	extraDependsReflectValue :=
		make([]reflect.Value, len(depends), len(depends))
	for i, depend := range depends {
		extraDependsReflectValue[i] = reflect.ValueOf(depend)
		if extraDependsReflectValue[i].Type().AssignableTo(rt) {
			return extraDependsReflectValue[i], loopTree, nil
		}
	}
	//
	var (
		found     = false
		value     reflect.Value
		valueType reflect.Type
	)
	for _, v := range f.factorys {
		if v.rv.Type().AssignableTo(rt) {
			found = true
			value = v.rv
			valueType = v.rt
		}
	}
	if !found {
		Err.Printf("Not found `%s` in Container! Auto register without initial!", rt)
		value = reflect.New(rt.Elem())
		valueType = rt
		f.RegisterFactory(value.Interface())
	}

	// Enject Fields
	for i := 0; i < valueType.Elem().NumField(); i++ {
		fv := value.Elem().Field(i)
		if !fv.CanSet() {
			continue
		}
		if fv.Type().Kind() == reflect.Ptr ||
			fv.Type().Kind() == reflect.Interface {
			found := false
			for j, depend := range extraDependsReflectValue {
				if depend.Type().AssignableTo(fv.Type()) {
					fv.Set(extraDependsReflectValue[j])
					found = true
					break
				}
			}
			if !found {
				factory, err := f.Lookup(fv.Type(), depends...)
				if err != nil {
					return emptyValue, loopTree, fmt.Errorf(
						"Can not Enject `%s.%s`,'%s'",
						valueType, fv, err.Error())
				}
				loopTree = append(loopTree, factory.Type().Name())
				fv.Set(factory)
			}
		}
	}

	return value, loopTree, nil
}

const (
	FactoryTypeStandalone = iota
	FactoryTypeStateful
	FactoryTypeStateless
)

type FactoryType int

// Factory is the struct that all the custom factory must be extended
// 其他自定义工厂必须集成自此结构
type Factory struct {
	state FactoryType
}

func (f *Factory) Type() FactoryType {
	return f.state
}
