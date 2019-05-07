package icalendar

import (
	"fmt"
	"github.com/heindl/caldav-go/icalendar/properties"
	"github.com/heindl/caldav-go/utils"
	"log"
	"reflect"
	"strings"
)

var _ = log.Print

func isInvalidOrEmptyValue(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}

func newValue(in reflect.Value) (out reflect.Value, isArrayElement bool) {

	typ := in.Type()
	kind := typ.Kind()

	for {
		if kind == reflect.Array || kind == reflect.Slice {
			isArrayElement = true
		} else if kind != reflect.Ptr {
			break
		}
		typ = typ.Elem()
		kind = typ.Kind()
	}

	out = reflect.New(typ)
	return

}

func dereferencePointerValue(v reflect.Value) reflect.Value {
	for (v.Kind() == reflect.Interface || v.Kind() == reflect.Ptr) && v.Elem().IsValid() {
		return v.Elem()
	}
	return v
}

func extractTagFromValue(v reflect.Value) (string, error) {

	vdref := dereferencePointerValue(v)
	vtemp, _ := newValue(vdref)

	if encoder, ok := vtemp.Interface().(properties.CanEncodeTag); ok {
		if tag, err := encoder.EncodeICalTag(); err != nil {
			return "", utils.NewError(extractTagFromValue, "unable to extract tag from interface", v.Interface(), err)
		} else {
			return strings.ToUpper(tag), nil
		}
	} else {
		typ := vtemp.Elem().Type()
		return strings.ToUpper(fmt.Sprintf("v%s", typ.Name())), nil
	}

}
