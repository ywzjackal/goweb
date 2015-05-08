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
	factorys []*factory
}

func (f *factoryContainer) Init() WebError {
	Log.Printf("INIT FC...")
	f.factorys = make([]*factory, 0)
	return nil
}

var emptyValue = reflect.ValueOf(0)

func (f *factoryContainer) Register(fac Factory) WebError {

	return nil
}

func (f *factoryContainer) Lookup(rt reflect.Type, ctx Context) (reflect.Value, WebError) {
	var (
		err    WebError      = nil // error
		target reflect.Value       // target which we are looking for
	)

	return target, err
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
