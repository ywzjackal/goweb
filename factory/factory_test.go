package factory

import (
	"testing"

	"github.com/ywzjackal/goweb"
)

type FactoryBase struct {
	FactoryStandalone
	Name string
}

func (f *FactoryBase) Init() {
	f.Name = "Facti"
	goweb.Log.Print("Init FactoryBase!")
}

type FactoryTest1 struct {
	FactoryStandalone
	Fac *FactoryBase
	Num int
}

func (f *FactoryTest1) Init() {
	f.Num = 2
	goweb.Log.Print("Init FactoryTest1!")
}

func Test_Factory(t *testing.T) {
	fc := NewContainer()
	fc.Register(&FactoryBase{}, "")
	fc.Register(&FactoryTest1{Num: 1}, "a")
	v, err := fc.LookupStandalone("a")
	if err != nil {
		t.Error("fail to look up factory!", err.ErrorAll())
		return
	}
	if !v.ReflectValue().IsValid() {
		t.Error("fail to look up factory valid!")
		return
	}
}

func Test_Factory2(t *testing.T) {
	fc := NewContainer()
	fc.Register(&FactoryBase{}, "")
	fc.Register(&FactoryTest1{Num: 2}, "a")
	a, err := fc.LookupStandalone("a")
	if err != nil {
		t.Error("Need look up factory, but it did not!", err.ErrorAll())
		return
	}
	if !a.ReflectValue().IsValid() {
		t.Error("Need look up a valid factory!")
		return
	}
	f := a.ReflectValue().Interface().(*FactoryTest1)
	if f.Num != 2 {
		t.Error("standalone factory got not as register value, exept 2, but got ", f.Num)
	}
}
