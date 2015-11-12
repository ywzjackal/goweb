package factory

import (
	"github.com/ywzjackal/goweb"
	"reflect"
)

type injectAble struct {
	targe goweb.Factory
	fType reflect.Type
	fValu reflect.Value
	stata []goweb.InjectNode
	statf []goweb.InjectNode
	statl []goweb.InjectNode
	fulnm string
	alias string
}

func (i *injectAble) Type() goweb.LifeType {
	return i.targe.Type()
}
func (i *injectAble) FullName() string                     { return i.fulnm }
func (i *injectAble) Alias() string                        { return i.alias }
func (i *injectAble) ReflectType() reflect.Type            { return i.fType }
func (i *injectAble) ReflectValue() reflect.Value          { return i.fValu }
func (i *injectAble) FieldsStandalone() []goweb.InjectNode { return i.stata }
func (i *injectAble) FieldsStateful() []goweb.InjectNode   { return i.statf }
func (i *injectAble) FieldsStateless() []goweb.InjectNode  { return i.statl }
func (i *injectAble) Target() goweb.Factory                { return i.targe }
