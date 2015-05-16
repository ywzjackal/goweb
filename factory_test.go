package goweb

import (
	"reflect"
	"testing"
)

type FactoryBase struct {
	FactoryStandalone
	Name string
}

func (f *FactoryBase) Init(){
	f.Name = "Facti"
	Log.Print("Init FactoryBase!")
}

type FactoryTest1 struct {
	FactoryStandalone
	Fac *FactoryBase
	Num int
}

func (f *FactoryTest1) Init() {
	f.Num = 2
	Log.Print("Init FactoryTest1!")
}

func Test_Factory(t *testing.T) {
	fc := &factoryContainer{}
	fc.Init()
	if err := fc.Register(&FactoryTest1{
		Num: 1,
	}); err != nil {
		t.Error(err.ErrorAll())
		return
	}
	if err := fc.Register(&FactoryTest1{
		Num: 9,
	}); err == nil {
		t.Error("Need return error when duplicate register factory!")
		return
	}
	v, err := fc.Lookup(reflect.TypeOf(&FactoryTest1{}), nil)
	if err != nil {
		t.Error("Need look up factory, but it did not!", err.ErrorAll())
		return
	}
	if !v.IsValid() {
		t.Error("Need look up a valid factory!")
		return
	}
	if !v.CanAddr() {
		t.Error("Need look up an addressable reflact.value! got %s", v)
		return
	}
	target := &FactoryTest1{
		Num: 3,
	}
	_v := reflect.ValueOf(target)
	for _v.Kind() == reflect.Ptr {
		_v = _v.Elem()
	}
	_v.Set(v)
	if target.Num != 2 {
		t.Error("Look up fail!")
	}
}

func Test_Factory2(t *testing.T) {
	fc := &factoryContainer{}
	fc.Init()
	
	v, err := fc.Lookup(reflect.TypeOf(&FactoryTest1{}), nil)
	if err != nil {
		t.Error("Need look up factory, but it did not!", err.ErrorAll())
		return
	}
	if !v.IsValid() {
		t.Error("Need look up a valid factory!")
		return
	}
	if !v.CanAddr() {
		t.Error("Need look up an addressable reflact.value! got %s", v)
		return
	}
	target := &FactoryTest1{
		Num: 3,
	}
	_v := reflect.ValueOf(target)
	for _v.Kind() == reflect.Ptr {
		_v = _v.Elem()
	}
	_v.Set(v)
	if target.Num != 2 {
		t.Error("Look up fail!")
	}
}