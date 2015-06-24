package dao

import (
	"reflect"
	"strings"
)

const (
	FieldTagName            = "name"
	FieldTagSelect          = "select"
	FieldTagAttribute       = "attribute"
	FieldTagValuePrimaryKey = "primarykey"
)

type attribute struct {
	reflect.StructTag
}

func (a attribute) Name() string {
	return a.Get(FieldTagName)
}

func (a attribute) Select() string {
	return a.Get(FieldTagSelect)
}

func (a attribute) IsPrimaryKey() bool {
	return a.attrHasKey(FieldTagValuePrimaryKey)
}

func (a attribute) attrHasKey(key string) bool {
	key = strings.ToLower(key)
	attrs := strings.ToLower(a.Get(FieldTagAttribute))
	values := strings.Split(attrs, ",")
	for _, v := range values {
		if v == key {
			return true
		}
	}
	return false
}
