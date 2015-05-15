package goweb

import (
	"reflect"
	"testing"
)

type FactoryTest1 struct {
	FactoryStandalone
	Num int
}

func Test_Factory(t *testing.T) {
	fc := &factoryContainer{}
	fc.Init()
	if err := fc.Register(&FactoryTest1{}); err != nil {
		t.Error(err)
		return
	}
	if err := fc.Register(&FactoryTest1{}); err == nil {
		t.Error("Need return error when duplicate register factory!")
		return
	}
	v, err := fc.Lookup(reflect.TypeOf(&FactoryTest1{}), nil)
	if err != nil {
		t.Error("Need look up factory, but it did not!", err.ErrorAll())
		return
	}
	if !v.CanAddr() {
		t.Error("Need look up an addressable reflact.value! got %s", v)
		return
	}
	_v := reflect.ValueOf(&FactoryTest1{})
	_v.Set(v)
}
