package dao

import (
	"testing"
	"time"
)

func Test_Hump2underscore(T *testing.T) {
	const src = "Test___AbcEfgOKabc"
	const tar = "test_abc_efg_ok_abc"
	temp := Hump2underscore(src)
	if temp != tar {
		T.Errorf("Except %s => %s, but got %s", src, tar, temp)
		return
	}
	T.Logf("Except %s => %s, got %s", src, tar, temp)
}

func Test_modelWrap_Init(T *testing.T) {
	const tableName = "tableName"
	type TestTable struct {
		Model
		Name  string    `name:"nameOfField" attribute:"primarykey"`
		Time  time.Time `name:"name_abc"`
		Pass  string
		Role  string
		Level int
	}
	m := modelWrap{}
	m.Init(&TestTable{}, tableName)
	for i, f := range m.Fields {
		T.Logf("[fields] key:%d, id:%d, name:%s, type:%s", i, f.Id, f.Name, f.Type)
	}
	for i, f := range m.Primarys {
		T.Logf("[PK] key:%d, id:%d, name:%s, type:%s", i, f.Id, f.Name, f.Type)
	}
	tar := m.NewTarget()
	for i, f := range tar.ScanInter {
		T.Logf("[scan] id:%d, %s", i, f)
	}
	T.Logf("%+v", tar.Inter)
}
