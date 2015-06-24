package dao

import (
	"reflect"
	"time"
)

type modelField struct {
	Name   string        // Name of field
	Select string        // Select value
	Type   reflect.Type  // Type of field
	Value  reflect.Value // value of field
	Inter  interface{}   // Interface of field
	Id     int           // Id of field
}

type modelWrap struct {
	Type      reflect.Type  // Type of Model
	Value     reflect.Value // Value of Model
	Name      string        // Name of Model
	TableName string        // Name of Table
	Fields    []*modelField // Fields of Model
	Primarys  []*modelField // Primary key Fields
	model     *Model
}

type modelTarget struct {
	Inter     interface{}   // Interface of target Model pointer
	ScanInter []interface{} // Scaner Interface of Model
}

func (m *modelWrap) Init(model interface{}, tableName string) {
	m.Type = reflect.TypeOf(model)
	m.Value = reflect.ValueOf(model)
	for m.Type.Kind() == reflect.Ptr {
		m.Type = m.Type.Elem()
	}
	for m.Value.Kind() == reflect.Ptr {
		m.Value = m.Value.Elem()
	}
	m.Name = m.Type.Name()
	m.TableName = tableName
	m.Fields = make([]*modelField, 0, m.Value.NumField())
	for i := 0; i < m.Value.NumField(); i++ {
		f := m.Value.Field(i)
		t := m.Type.Field(i)
		if !f.CanSet() {
			continue
		}
		switch f.Type().Kind() {
		case reflect.Ptr:
			panic("Unsuport Model Field Type:" + f.Type().Kind().String())
		case reflect.Struct:
			switch typ := f.Interface().(type) {
			case time.Time:
			case Model:
				m.model = &typ
				continue
			default:
				panic("Unsuport Model Field Type:" + f.Type().String())
			}
		case reflect.Slice:
			panic("Unsuport Model Field Type:" + f.Type().Kind().String())
		case reflect.Map:
			panic("Unsuport Model Field Type:" + f.Type().Kind().String())
		case reflect.Interface:
			panic("Unsuport Model Field Type:" + f.Type().Kind().String())
		default:
		}
		attr := attribute{t.Tag}
		mf := &modelField{}
		mf.Id = i
		mf.Inter = f.Interface()
		mf.Name = attr.Name()
		if mf.Name == "" {
			mf.Name = Hump2underscore(t.Name)
		}
		mf.Select = attr.Select()
		mf.Type = f.Type()
		mf.Value = f
		m.Fields = append(m.Fields, mf)
		if attr.IsPrimaryKey() {
			m.Primarys = append(m.Primarys, mf)
		}
	}
}

func (m *modelWrap) NewTarget() *modelTarget {
	var (
		rv  reflect.Value = reflect.New(m.Type)
		tar interface{}   = rv.Interface()
		scn []interface{} = make([]interface{}, len(m.Fields), len(m.Fields))
	)

	for i, f := range m.Fields {
		scn[i] = rv.Elem().Field(f.Id).Addr()
	}
	return &modelTarget{
		Inter:     tar,
		ScanInter: scn,
	}
}

const gap = 'A' - 'a'

func Hump2underscore(str string) string {
	var tar string = ""
	var pre_underscore = true
	var pre_capital = false
	var pre_multi_capital = false
	for _, char := range str {
		switch {
		case char == '_':
			if !pre_underscore {
				tar += "_"
			}
			pre_underscore = true
			pre_capital = false
			pre_multi_capital = false
		case char >= 'A' && char <= 'Z':
			if pre_capital {
				pre_multi_capital = true
			} else {
				pre_multi_capital = false
			}
			if !pre_underscore && !pre_capital {
				tar += "_"
			} else {
				pre_underscore = false
			}
			pre_capital = true
			tar += string(char - gap)
		default:
			if pre_multi_capital {
				tar += "_"
			}
			tar += string(char)
			pre_underscore = false
			pre_capital = false
			pre_multi_capital = false
		}
	}
	return tar
}
