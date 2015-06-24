package dao

import (
	"testing"
	"time"
)

func Test_Postgresql(T *testing.T) {
	type TestModel struct {
		Model
		Name string
		Pass string
		Time time.Time `name:"timestamp" select:"timestamp(\"timestamp\")"`
	}
	pg := newDaoPostgres(nil, "tablename", &TestModel{})
	T.Log(pg.qsFindById)
	T.Log(pg.qsFindAll)
}
