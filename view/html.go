package view

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/howeyc/fsnotify"
	"github.com/ywzjackal/goweb"
	"io"
	"path/filepath"
	"sync"
)

type ViewHtml struct {
	template *template.Template
	watcher  *fsnotify.Watcher
	sync.Mutex
	io.Closer
}

func NewViewHtml(dirPath, suffix, delimsLeft, delimsRight string, funcMap template.FuncMap) *ViewHtml {
	var (
		err     error
		pattern string
	)
	dirPath, err = filepath.Abs(dirPath)
	if err != nil {
		panic(err)
	}
	pattern = dirPath + "/" + suffix
	view := &ViewHtml{}
	view.template = template.Must(template.New("").Funcs(funcMap).Delims(delimsLeft, delimsRight).ParseGlob(pattern))
	view.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			select {
			case ev := <-view.watcher.Event:
				view.Lock()
				tmpls, err := template.New("").Funcs(funcMap).Delims(delimsLeft, delimsRight).ParseGlob(pattern)
				if err != nil {
					goweb.Err.Println("fail to reload templates with ", dirPath, " because ", err)
				} else {
					view.template = tmpls
					goweb.Log.Println("reload templates with ", dirPath, " because ", ev)
				}
				view.Unlock()
			case err := <-view.watcher.Error:
				goweb.Err.Println("error:", err)
			}
		}
	}()
	err = view.watcher.Watch(dirPath)
	if err != nil {
		goweb.Err.Println("fail watch ", dirPath, ", ", err)
	}
	return view
}

func (v *ViewHtml) Render(name string, writer io.Writer, model interface{}) (err goweb.WebError) {
	defer func() {
		v.Unlock()
		if r := recover(); r != nil {
			writer.Write([]byte(fmt.Sprintf("%v", r)))
		}
	}()
	v.Lock()
	e := v.template.ExecuteTemplate(writer, name, model)
	if e != nil {
		return goweb.NewWebError(http.StatusInternalServerError, e.Error())
	}
	return err
}

func (v *ViewHtml) Close() error {
	v.Lock()
	v.watcher.Close()
	v.Unlock()
	return nil
}
