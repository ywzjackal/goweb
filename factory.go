package goweb

import (
	"fmt"
	"reflect"
)

type FactoryContainer interface {
	Init() error
	RegisterFactory(interface{})
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
	if t.Kind() != reflect.Ptr {
		panic(fmt.Errorf("RegisterFactory must be a Pointer of Factory! got %s",
			t.Kind()))
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
	}
	for _, v := range f.factorys {
		if v.rv.Type().AssignableTo(rt) {
			// Enject
			for i := 0; i < v.rt.Elem().NumField(); i++ {
				fv := v.rv.Elem().Field(i)
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
						//					if fv.IsNil() {
						factory, err := f.Lookup(fv.Type(), depends...)
						if err != nil {
							return emptyValue, loopTree, fmt.Errorf(
								"Can not Enject `%s.%s`,'%s'",
								v.rt, fv, err.Error())
						}
						loopTree = append(loopTree, factory.Type().Name())
						fv.Set(factory)
						//					} else {
						//						Log("Not Nil")
						//					}
					}
				} else {
					return emptyValue, loopTree,
						fmt.Errorf("%s(%s) is not a pointer or interface",
							fv.Type().Name(), fv.Type().Kind())
				}
			}
			return v.rv, loopTree, nil
		}
	}
	for _, depend := range extraDependsReflectValue {
		if depend.Type().AssignableTo(rt) {
			return depend, loopTree, nil
		}
	}
	e := fmt.Errorf("Not found `%s` in Container!", rt)
	return emptyValue, loopTree, e
}

type FactoryInterface interface {
}

type Factory struct {
	FactoryInterface
}
