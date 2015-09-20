package view

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/ywzjackal/goweb"
	"io"
)

type ViewHtml struct {
	*template.Template
}

func NewViewHtml(dirPath, delimsLeft, delimsRight string, funcMap template.FuncMap) ViewHtml {
	return ViewHtml{
		template.Must(template.New("").Funcs(funcMap).Delims(delimsLeft, delimsRight).ParseGlob(dirPath)),
	}
}

func (v *ViewHtml) Render(name string, writer io.Writer, model interface{}) (err goweb.WebError) {
	defer func() {
		if r := recover(); r != nil {
			writer.Write([]byte(fmt.Sprintf("%v", r)))
		}
	}()

	e := v.ExecuteTemplate(writer, name, model)
	if e != nil {
		return goweb.NewWebError(http.StatusInternalServerError, e.Error())
	}
	return err
}
