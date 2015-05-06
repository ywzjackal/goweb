package goweb

import (
	"fmt"
	"runtime"
)

type WebErrorStackNode struct {
	File string
	Line int
	Func string
}

type WebError interface {
	error
	ErrorAll() string
	Code() int
	File() string
	Line() int
	FuncName() string
	Children() []WebError
	CallStack() []WebErrorStackNode
	Append(code int, msg string, args ...interface{}) WebError
}

type gowebError struct {
	WebError
	code     int
	msg      string
	children []WebError
	file     string
	line     int
	funcname string
	stack    []WebErrorStackNode
}

func NewWebError(code int, msg string, args ...interface{}) WebError {
	return newWebError(true, code, fmt.Sprintf(msg, args...))
}

func newWebError(needStack bool, code int, msg string) *gowebError {
	// if this is end of Error Stack, collect stack of this error
	stack := []WebErrorStackNode{}
	if needStack {
		for i := 2; ; i++ {
			ptr, file, line, ok := runtime.Caller(i)
			funcname := runtime.FuncForPC(ptr).Name()
			if !ok {
				break
			}
			stack = append(stack, WebErrorStackNode{file, line, funcname})
		}
	}
	ptr, file, line, callok := runtime.Caller(2)
	funcname := ""
	if callok {
		funcname = runtime.FuncForPC(ptr).Name()
	}
	err := &gowebError{
		code:     code,
		msg:      msg,
		file:     file,
		line:     line,
		funcname: funcname,
		stack:    stack,
	}
	err.children = []WebError{err}
	return err
}

func (e *gowebError) Code() int {
	return e.code
}

func (e *gowebError) Error() string {
	return e.msg
}

func (e *gowebError) ErrorAll() string {
	short := e.file
	for i := len(e.file) - 1; i > 0; i-- {
		if e.file[i] == '/' {
			short = e.file[i+1:]
			break
		}
	}
	return fmt.Sprintf("%s:%d %s", short, e.line, e.msg)
}

func (e *gowebError) Children() []WebError {
	return e.children
}

func (e *gowebError) CallStack() []WebErrorStackNode {
	return e.stack
}

func (e *gowebError) Append(code int, msg string, args ...interface{}) WebError {
	err := newWebError(false, code, fmt.Sprintf(msg, args...))
	e.children = append(e.children, err)
	return e
}

func (e *gowebError) File() string {
	return e.file
}

func (e *gowebError) Line() int {
	return e.line
}

func (e *gowebError) FuncName() string {
	return e.funcname
}
