package router

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/ywzjackal/goweb"
	"github.com/ywzjackal/goweb/view"
)

const (
	__router_name = "goweb.router"
)

var (
	DefaultControllerName = "Default"
	DefaultActionName     = "Default"
	__gloabl_goweb_router = &router{}
)

type ContextGenerator func(http.ResponseWriter, *http.Request) goweb.Context

type router struct {
	http.Handler
	controllers goweb.ControllerContainer
	ctxGetor    ContextGenerator
}

func NewRouter(
	c goweb.ControllerContainer,
	ctxGetor ContextGenerator,
) goweb.Router {
	return &router{
		controllers: c,
		ctxGetor:    ctxGetor,
	}
}

func (r *router) Name() string {
	return __router_name
}

func (r *router) ControllerContainer() goweb.ControllerContainer {
	return r.controllers
}

func (r *router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var (
		begin     = time.Now()
		urlprefix = strings.ToLower(req.URL.Path)
		ctx       = r.ctxGetor(res, req)
		rts       []reflect.Value
		err       goweb.WebError
		_err      goweb.WebError
	)

	ctl, err := r.controllers.Get(urlprefix, ctx)
	if err != nil || ctl == nil {
		ctx.SetError(err)
		goto ERROR_USER_REPORT
	} else {
		rts, err = ctl.Call(req.Method, ctx)
		if err == nil {
			if ctx.Error() == nil || ctx.Error().Code() == 0 {
				err = render(rts, ctl)
				if err != nil {
					goto ERROR_USER_REPORT
				}
			} else {
				err = ctx.Error()
				goto ERROR_USER_REPORT
			}
			goto FINISH
		} else {
			err.Append(http.StatusMethodNotAllowed, "Fail to call `%s`->`%s`", ctl, req.Method)
			ctx.SetError(err)
			goto ERROR_USER_REPORT
		}
	}
	goto FINISH

FINISH:
	if goweb.Debug {
		goweb.Log.Printf("%s: %s %d %dus", req.Method, req.URL.Path, 200, time.Now().Sub(begin).Nanoseconds()/1000)
	}
	return

ERROR_USER_REPORT:
	ctx.Request().URL.Path = fmt.Sprintf("/%d", err.Code())
	ctl, _err = r.controllers.Get(ctx.Request().URL.Path, ctx)
	if _err == nil && ctl != nil {
		rts, _err = ctl.Call(req.Method, ctx)
		if _err == nil {
			_err = render(rts, ctl)
		} else {
			goto DEFAULT_ERROR_USER_REPORT
		}
	} else {
		goto DEFAULT_ERROR_USER_REPORT
	}
	goto FINISH

DEFAULT_ERROR_USER_REPORT:
	ctx.Request().URL.Path = "/error"
	ctl, _err = r.controllers.Get(ctx.Request().URL.Path, ctx)
	if _err == nil && ctl != nil {
		rts, _err = ctl.Call(req.Method, ctx)
		if _err == nil {
			_err = render(rts, ctl)
			if _err != nil {
				err = _err
				goto DEFAULT_ERROR_REPORT
			}
		} else {
			goto DEFAULT_ERROR_REPORT
		}
	} else {
		goto DEFAULT_ERROR_REPORT
	}
	goto FINISH

DEFAULT_ERROR_REPORT:
	res.WriteHeader(err.Code())
	res.Write([]byte("<!DOCTYPE html>\r\n<html><head><title>HTTP-Internal Server Error</title></head>"))
	res.Write([]byte(fmt.Sprintf("<body><h1>%d : %s</h1><hr/>", err.Code(), http.StatusText(err.Code()))))
	res.Write([]byte(fmt.Sprintf("<h3>Error stack:</h3><ul>")))
	for _, _err := range err.Children() {
		res.Write([]byte(fmt.Sprintf("<li>%s</li>", _err.ErrorAll())))
	}
	res.Write([]byte(fmt.Sprintf("</ul><h3>Call stack:</h3><ul>")))
	for _, _nod := range err.CallStack() {
		res.Write([]byte(fmt.Sprintf("<li>%s:%d</li>", _nod.Func, _nod.Line)))
	}
	res.Write([]byte("</ul><hr><h4>Power by GoWeb github.com/ywzjackal/goweb </h4></body></html>"))
	goto FINISH
}

func render(rets []reflect.Value, c goweb.Controller) goweb.WebError {
	if len(rets) == 0 {
		return goweb.NewWebError(1, "Controller Action need return a ViewType like `html`,`json`.")
	}
	viewType, ok := rets[0].Interface().(string)
	if !ok {
		return goweb.NewWebError(1, "Controller Action need return a ViewType of string! but got `%s`", rets[0].Type())
	}
	view := view.GetView(viewType)
	if view == nil {
		return goweb.NewWebError(1, "Unknow ViewType :%s", viewType)
	}
	interfaces := make([]interface{}, len(rets)-1, len(rets)-1)
	for i, ret := range rets[1:] {
		interfaces[i] = ret.Interface()
	}
	err := view.Render(c, interfaces...)
	if err != nil {
		return err.Append(500, "Fail to render view %s, data:%+v", viewType, interfaces)
	}
	return nil
}
