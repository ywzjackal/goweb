package view

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/ywzjackal/goweb"
)

var (
	// TemplateSuffix is template suffix -.-
	TemplateSuffix = ".html"
	// TemplatePosition is template position path -.-
	TemplatePosition = "" //"./templates"
	// html.template delims attributes of left
	DelimsLeft = "{{"
	// html.template delims attributes of right
	DelimsRight = "}}"
	// views map container
	views = make(map[string]goweb.View)
	// never mind! -.-
	rootTemplate = template.New("")
	//
	TemplateFuncs = template.FuncMap{
		"httpStatusText": http.StatusText,
	}
)

// Initialize buildin view components
func init() {
	RegisterView("html", &viewHtml{})
	RegisterView("json", &viewJson{})
	RegisterView("", &view{})
	//
	//	ReloadTemplates()
}

// ReloadTemplates, you know what it will do. `.`
func ReloadTemplates() {
	if TemplatePosition == "" {
		panic("Please set `goweb.TemplatePosition to path of templates directory!`")
	}
	rootTemplate = template.New("").Funcs(TemplateFuncs)
	rootTemplate = template.Must(rootTemplate.Delims(DelimsLeft, DelimsRight).
		ParseGlob(TemplatePosition + "/*"))
}

// RegisterView should be called by custom view component in 'package file init() function'
// will panic when register with duplicate name.
//
// RegisterView 应该在用户引用的自定义视图组件的包文件的init（）函数中调用以注册新的视图组件
// 如果出现panic，说明视图组件的名字被重复注册
func RegisterView(name string, view goweb.View) {
	if _, ok := views[strings.ToLower(name)]; ok {
		panic("Register view `" + name + "` duplicate!")
	}
	views[strings.ToLower(name)] = view
}

func GetView(name string) goweb.View {
	v, exist := views[name]
	if exist {
		return v
	}
	return nil
}

type view struct {
	goweb.View
}

type ViewHtml interface {
	goweb.View
}

type ViewJson interface {
	goweb.View
}

type viewHtml struct {
	ViewHtml
	*view
	//
	req *http.Request
}

type viewJson struct {
	ViewJson
	*view
}

func (v *view) Render(c goweb.Controller, args ...interface{}) goweb.WebError {
	//	raw := []byte(fmt.Sprintf("% +v, % +v", c, args))
	//	_, err := c.Context().ResponseWriter().Write(raw)
	//	if err != nil {
	//		return goweb.NewWebError(500, err.Error())
	//	}
	return nil
}

func (v *viewHtml) Render(c goweb.Controller, args ...interface{}) (err goweb.WebError) {
	defer func() {
		if r := recover(); r != nil {
			c.Context().ResponseWriter().Write([]byte(fmt.Sprintf("%v", r)))
		}
	}()
	var (
		name = strings.ToLower(c.Context().Request().URL.Path)
	)
	if goweb.Debug {
		ReloadTemplates()
	}
	buffer := bytes.Buffer{}
	writer := bufio.NewWriter(&buffer)
	switch len(args) {
	case 0:
		e := rootTemplate.ExecuteTemplate(writer, name, c)
		if e != nil {
			return goweb.NewWebError(500, e.Error())
		}
	case 1:
		name, ok := args[0].(string)
		if !ok {
			return goweb.NewWebError(500, "invalid view template name:%+v,need string", args[0])
		}
		e := rootTemplate.ExecuteTemplate(writer, name, c)
		if e != nil {
			return goweb.NewWebError(500, e.Error())
		}
	default:
		e := rootTemplate.ExecuteTemplate(writer, name, c)
		if e != nil {
			return goweb.NewWebError(500, e.Error())
		}
	}
	writer.Flush()
	c.Context().ResponseWriter().Header().Add("Cache-Control", "no-store, must-revalidate")
	c.Context().ResponseWriter().Header().Add("Pragma", "no-cache")
	c.Context().ResponseWriter().Write(buffer.Bytes())
	return err
}

func (v *viewJson) Render(c goweb.Controller, args ...interface{}) goweb.WebError {
	var (
		b   []byte = nil
		err error  = nil
	)
	switch len(args) {
	case 0:
		b, err = json.MarshalIndent(c, "", " ")
	case 1:
		b, err = json.MarshalIndent(args[0], "", " ")
	default:
		b, err = json.MarshalIndent(args, "", " ")
	}
	if err != nil {
		return goweb.NewWebError(500, err.Error())
	}
	c.Context().ResponseWriter().Header().Add("Cache-Control", "no-store, must-revalidate")
	c.Context().ResponseWriter().Header().Add("Pragma", "no-cache")
	_, err = c.Context().ResponseWriter().Write(b)
	return nil
}
