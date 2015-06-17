package goweb

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"
)

const (
	__router_name = "goweb.router"
)

var (
	DefaultControllerName = "Default"
	DefaultActionName     = "Default"
	__gloabl_goweb_router = &router{}
)

func init() {
	RegisterRouterCreator(__router_name, __gloabl_goweb_router.New)
}

type router struct {
	http.Handler
	controllers ControllerContainer
	factorys    FactoryContainer
	storage     Storage
}

func (r *router) New(scope string, args ...interface{}) Router {
	router := &router{}
	for _, arg := range args {
		switch v := arg.(type) {
		case ControllerContainer:
			router.controllers = v
		case FactoryContainer:
			router.factorys = v
		case Storage:
			router.storage = v
		default:
			panic(fmt.Sprintf("Invalid argument to create router! %s", arg))
		}
	}
	if router.storage == nil {
		panic("need set an memory storage to router!")
	}
	if router.controllers == nil {
		panic("need set an controllers container to router!")
	}
	if router.factorys == nil {
		panic("need set an factorys container to router!")
	}

	return router
}

func (r *router) Name() string {
	return __router_name
}

func (r *router) Init() WebError {
	if err := r.MemStorage().Init(); err != nil {
		return err
	}
	if err := r.FactoryContainer().Init(); err != nil {
		return err
	}
	if err := r.ControllerContainer().Init(r.FactoryContainer()); err != nil {
		return err
	}
	return nil
}

func (r *router) ControllerContainer() ControllerContainer {
	return r.controllers
}

func (r *router) FactoryContainer() FactoryContainer {
	return r.factorys
}

func (r *router) MemStorage() Storage {
	return r.storage
}

func (r *router) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	var (
		begin     = time.Now()
		urlprefix = strings.ToLower(req.URL.Path)
		session   = &session{}
		ctx       = &context{
			request:          req,
			responseWriter:   res,
			factoryContainer: r.FactoryContainer(),
			session:          session,
		}
		rts  []reflect.Value
		err  WebError
		_err WebError
	)

	session.Init(res, req, r.storage)

	ctl, err := r.controllers.Get(urlprefix, ctx)
	if err != nil || ctl == nil {
		ctx.err = err
		goto ERROR_USER_REPORT
	} else {
		rts, err = ctl.Call(req.Method, ctx)
		if err == nil {
			err = render(rts, ctl)
			if err != nil {
				goto ERROR_USER_REPORT
			}
			goto FINISH
		} else {
			err.Append(http.StatusMethodNotAllowed, "Fail to call `%s`->`%s`", ctl, req.Method)
			ctx.err = err
			goto ERROR_USER_REPORT
		}
	}
	goto FINISH

FINISH:
	if Debug {
		Log.Printf("%s: %s %d %dus", req.Method, req.URL.Path, 200, time.Now().Sub(begin).Nanoseconds()/1000)
	}
	return

ERROR_USER_REPORT:
	ctx.request.URL.Path = fmt.Sprintf("/%d", err.Code())
	ctl, _err = r.controllers.Get(ctx.request.URL.Path, ctx)
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
	ctx.request.URL.Path = "/error"
	ctl, _err = r.controllers.Get(ctx.request.URL.Path, ctx)
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
