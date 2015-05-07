package goweb

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

var (
	TemplateSuffix   = ".html"
	TemplatePosition = "undefined!" //"./templates"
	DelimsLeft       = "{{"
	DelimsRight      = "}}"
	views            = make(map[string]View)
	rootTemplate     = template.New("")
)

func init() {
	Log.Print("INIT VIEWS...")
	views["html"] = &viewHtml{}
	views["json"] = &ViewJson{}
	views[""] = &view{}
	//
	ReloadTemplates()
}

func ReloadTemplates() {
	if TemplatePosition == "undefined!" {
		return
	}
	rootTemplate = template.New("")
	rootTemplate = template.Must(rootTemplate.Delims(DelimsLeft, DelimsRight).
		ParseGlob(TemplatePosition + "/*"))
}

func RegisterView(name string, view View) {
	views[name] = view
}

type View interface {
	Render(Controller2, ...interface{}) WebError
	ResponseWriter() http.ResponseWriter
}

type view struct {
	View
	res http.ResponseWriter
}

type ViewHtml interface {
	Request() *http.Request
}

type viewHtml struct {
	ViewHtml
	*view
	//
	req *http.Request
}

type ViewJson struct {
	*view
}

func (v *view) Render(c Controller2, args ...interface{}) WebError {
	raw := []byte(fmt.Sprintf("% +v, % +v", c, args))
	_, err := c.Context().ResponseWriter().Write(raw)
	return NewWebError(1, err.Error())
}

func (v *view) ResponseWriter() http.ResponseWriter {
	return v.res
}

func (v *viewHtml) Request() *http.Request {
	return v.req
}

func (v *viewHtml) Render(c Controller2, args ...interface{}) WebError {
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

func (v *ViewJson) Render(c Controller2, args ...interface{}) WebError {
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
