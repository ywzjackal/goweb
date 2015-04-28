package goweb

import (
	"fmt"
)

type Error interface {
	Code() int
	Msg() string
}

type gowebError struct {
	Error
	code int
	msg  string
}

func NewError(code int, msg string) Error {
	return &gowebError{
		code: code,
		msg:  msg,
	}
}

func (e *gowebError) String() string {
	return fmt.Sprintf("Error %d!\r\n%s\r\n", e.code, e.msg)
}

func (e *gowebError) Code() int {
	return e.code
}

func (e *gowebError) Msg() string {
	return e.msg
}
