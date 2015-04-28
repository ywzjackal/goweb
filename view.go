package goweb

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

var (
	TemplateSuffix   = ".html"
	TemplatePosition = "./templates"
	views            = make(map[string]View)
	rootTemplate     = template.New("")
)

func init() {
	Log.Print("INIT VIEWS...")
	views["html"] = &viewHtml{}
	views["json"] = &ViewJson{}
	//
	rootTemplate = template.Must(rootTemplate.Delims("{{{", "}}}").ParseGlob(TemplatePosition + "/*"))
}

type View interface {
	Render(Context, ...interface{}) error
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

func (v *view) Render(c Context, name string, args ...interface{}) error {
	raw := []byte(fmt.Sprintf("%s:% +v", name, args))
	_, err := c.ResponseWriter().Write(raw)
	return err
}

func (v *view) ResponseWriter() http.ResponseWriter {
	return v.res
}

func (v *viewHtml) Request() *http.Request {
	return v.req
}

func (v *viewHtml) Render(c Context, args ...interface{}) error {
	var (
		name = strings.ToLower(c.ControllerName() + "_" + c.ActionName())
		err  error
	)
	if len(args) > 0 && args[0] != nil {
		rootTemplate.ExecuteTemplate(c.ResponseWriter(), name, args[0])
	} else {
		rootTemplate.ExecuteTemplate(c.ResponseWriter(), name, nil)
	}
	return err
}

func (v *ViewJson) Render(c Context, args ...interface{}) error {
	var err error
	if len(args) == 1 {
		b, err := json.MarshalIndent(args[0], "", " ")
		if err != nil {
			return err
		}
		_, err = c.ResponseWriter().Write(b)
	} else {
		b, err := json.MarshalIndent(args, "", " ")
		if err != nil {
			return err
		}
		_, err = c.ResponseWriter().Write(b)
	}
	return err
}
