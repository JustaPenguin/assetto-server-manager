package utils

import (
	"fmt"
	"reflect"
	"runtime"
)

type Error struct {
	method  interface{}
	message string
	context interface{}
	cause   error
}

func NewError(method interface{}, message string, context interface{}, cause error) *Error {
	e := new(Error)
	e.method = method
	e.message = message
	e.context = context
	e.cause = cause
	return e
}

func (e *Error) Error() string {
	pc := reflect.ValueOf(e.method).Pointer()
	fn := runtime.FuncForPC(pc).Name()
	msg := fmt.Sprintf("error: %s\nfunc: %s", e.message, fn)
	if e.context != nil {
		tname := reflect.ValueOf(e.context).Type()
		msg = fmt.Sprintf("%s\ncontext: %s", msg, tname.String())
	}
	if e.cause != nil {
		msg = fmt.Sprintf("%s\ncause: %s", msg, e.cause.Error())
	}
	return msg
}
