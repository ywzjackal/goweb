package goweb

import (
	"bufio"
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"
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
	views = make(map[string]View)
	// never mind! -.-
	rootTemplate = template.New("")
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
	rootTemplate = template.New("")
	rootTemplate = template.Must(rootTemplate.Delims(DelimsLeft, DelimsRight).
		ParseGlob(TemplatePosition + "/*"))
}

// RegisterView should be called by custom view component in 'package file init() function'
// will panic when register with duplicate name.
//
// RegisterView 应该在用户引用的自定义视图组件的包文件的init（）函数中调用以注册新的视图组件
// 如果出现panic，说明视图组件的名字被重复注册
func RegisterView(name string, view View) {
	if _, ok := views[strings.ToLower(name)]; ok {
		panic("Register view `" + name + "` duplicate!")
	}
	views[strings.ToLower(name)] = view
}

// View is the top of view component's interface, all custom view component need
// implament from this, and realize method Render(Controller, ...interface{}) WebError
//
// View 是视图的定级接口组件，所有的自定义视图组件必须实现此接口
type View interface {
	Render(Controller, ...interface{}) WebError
}

type view struct {
	View
}

type ViewHtml interface {
	View
}

type ViewJson interface {
	View
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

func (v *view) Render(c Controller, args ...interface{}) WebError {
	//	raw := []byte(fmt.Sprintf("% +v, % +v", c, args))
	//	_, err := c.Context().ResponseWriter().Write(raw)
	//	if err != nil {
	//		return NewWebError(500, err.Error())
	//	}
	return nil
}

func (v *viewHtml) Render(c Controller, args ...interface{}) WebError {
	var (
		name          = strings.ToLower(c.Context().Request().URL.Path)
		err  WebError = nil
	)
	if Debug {
		ReloadTemplates()
	}
	buffer := bytes.Buffer{}
	writer := bufio.NewWriter(&buffer)
	switch len(args) {
	case 0:
		e := rootTemplate.ExecuteTemplate(writer, name, c)
		if e != nil {
			return NewWebError(500, e.Error())
		}
	case 1:
		name, ok := args[0].(string)
		if !ok {
			return NewWebError(500, "invalid view template name:%+v,need string", args[0])
		}
		e := rootTemplate.ExecuteTemplate(writer, name, c)
		if e != nil {
			return NewWebError(500, e.Error())
		}
	default:
		e := rootTemplate.ExecuteTemplate(writer, name, c)
		if e != nil {
			return NewWebError(500, e.Error())
		}
	}
	writer.Flush()
	c.Context().ResponseWriter().WriteHeader(200)
	c.Context().ResponseWriter().Write(buffer.Bytes())
	return err
}

func (v *viewJson) Render(c Controller, args ...interface{}) WebError {
	if len(args) == 1 {
		b, err := json.MarshalIndent(c, "", " ")
		if err != nil {
			return NewWebError(500, err.Error())
		}
		_, err = c.Context().ResponseWriter().Write(b)
	} else {
		b, err := json.MarshalIndent(args, "", " ")
		if err != nil {
			return NewWebError(500, err.Error())
		}
		_, err = c.Context().ResponseWriter().Write(b)
	}
	return nil
}
