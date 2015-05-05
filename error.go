package goweb

import (
	"fmt"
	"runtime"
)

type WebError interface {
	Code() int
	Error() string
	Child() WebError
	ErrorAllText() string
}

type gowebError struct {
	WebError
	code     int
	msg      string
	child    WebError
	file     string
	line     int
	callerok bool
}

func NewWebError(code int, msg string, child WebError) WebError {
	_, file, line, ok := runtime.Caller(1)
	return &gowebError{
		code:     code,
		msg:      msg,
		child:    child,
		file:     file,
		line:     line,
		callerok: ok,
	}
}

func (e *gowebError) Code() int {
	return e.code
}

func (e *gowebError) Error() string {
	short := e.file
	for i := len(e.file) - 1; i > 0; i-- {
		if e.file[i] == '/' {
			short = e.file[i+1:]
			break
		}
	}
	return fmt.Sprintf("%s:%d %s", short, e.line, e.msg)
}

func (e *gowebError) Child() WebError {
	return e.child
}

func (e *gowebError) ErrorAllText() string {
	txt := ""
	txt = e.Error() + " \r\n"
	for err := e.Child(); err != nil; {
		txt = txt + fmt.Sprintf("\t %s \r\n", err.Error())
	}
	return txt
}
