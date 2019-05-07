package icalendar

import (
	"fmt"
	"github.com/heindl/caldav-go/icalendar/properties"
	"github.com/heindl/caldav-go/utils"
	"log"
	"reflect"
	"strings"
)

const (
	Newline = "\r\n"
)

var _ = log.Print

type encoder func(reflect.Value) (string, error)

func tagAndJoinValue(v reflect.Value, in []string) (string, error) {
	if tag, err := extractTagFromValue(v); err != nil {
		return "", utils.NewError(tagAndJoinValue, "unable to extract tag from value", v, err)
	} else {
		var out []string
		out = append(out, properties.MarshalProperty(properties.NewProperty("begin", tag)))
		out = append(out, in...)
		out = append(out, properties.MarshalProperty(properties.NewProperty("end", tag)))
		return strings.Join(out, Newline), nil
	}
}

func marshalCollection(v reflect.Value) (string, error) {

	var out []string

	for i, n := 0, v.Len(); i < n; i++ {
		vi := v.Index(i).Interface()
		if encoded, err := Marshal(vi); err != nil {
			msg := fmt.Sprintf("unable to encode interface at index %d", i)
			return "", utils.NewError(marshalCollection, msg, vi, err)
		} else if encoded != "" {
			out = append(out, encoded)
		}
	}

	return strings.Join(out, Newline), nil

}

func marshalStruct(v reflect.Value) (string, error) {

	var out []string

	// iterate over all fields
	vtype := v.Type()
	n := vtype.NumField()

	for i := 0; i < n; i++ {

		// keep a reference to the field value and definition
		fv := v.Field(i)
		fs := vtype.Field(i)

		// use the field definition to extract out property defaults
		p := properties.PropertyFromStructField(fs)
		if p == nil {
			continue // skip explicitly ignored fields and private members
		}

		fi := fv.Interface()

		// some fields are not properties, but actually nested objects.
		// detect those early using the property and object encoder...
		if _, ok := fi.(properties.CanEncodeValue); !ok && !isInvalidOrEmptyValue(fv) {
			if encoded, err := encode(fv, objectEncoder); err != nil {
				msg := fmt.Sprintf("unable to encode field %s", fs.Name)
				return "", utils.NewError(marshalStruct, msg, v.Interface(), err)
			} else if encoded != "" {
				// encoding worked! no need to process as a property
				out = append(out, encoded)
				continue
			}
		}

		// now check to see if the field value overrides the defaults...
		if !isInvalidOrEmptyValue(fv) {
			// first, check the field value interface for overrides...
			if overrides, err := properties.PropertyFromInterface(fi); err != nil {
				msg := fmt.Sprintf("field %s failed validation", fs.Name)
				return "", utils.NewError(marshalStruct, msg, v.Interface(), err)
			} else if p.Merge(overrides); p.Value == "" {
				// then, if we couldn't find an override from the interface,
				// try the simple string encoder...
				if p.Value, err = stringEncoder(fv); err != nil {
					msg := fmt.Sprintf("unable to encode field %s", fs.Name)
					return "", utils.NewError(marshalStruct, msg, v.Interface(), err)
				}
			}
		}

		// make sure we have a value by this point
		if !p.HasNameAndValue() {
			if p.OmitEmpty {
				continue
			} else if p.DefaultValue != "" {
				p.Value = p.DefaultValue
			} else if p.Required {
				msg := fmt.Sprintf("missing value for required field %s", fs.Name)
				return "", utils.NewError(Marshal, msg, v.Interface(), nil)
			}
		}

		// encode in the property
		out = append(out, properties.MarshalProperty(p))

	}

	// wrap the fields in the enclosing struct tags
	return tagAndJoinValue(v, out)

}

func objectEncoder(v reflect.Value) (string, error) {

	// decompose the value into its interface parts
	v = dereferencePointerValue(v)

	// encode the value based off of its type
	switch v.Kind() {
	case reflect.Slice:
		fallthrough
	case reflect.Array:
		return marshalCollection(v)
	case reflect.Struct:
		return marshalStruct(v)
	}

	return "", nil

}

func stringEncoder(v reflect.Value) (string, error) {
	return fmt.Sprintf("%v", v.Interface()), nil
}

func propertyEncoder(v reflect.Value) (string, error) {

	vi := v.Interface()
	if p, err := properties.PropertyFromInterface(vi); err != nil {

		// return early if interface fails its own validation
		return "", err

	} else if p.HasNameAndValue() {

		// if an interface encodes its own name and value, it's a property
		return properties.MarshalProperty(p), nil

	}

	return "", nil

}

func encode(v reflect.Value, encoders ...encoder) (string, error) {

	for _, encode := range encoders {
		if encoded, err := encode(v); err != nil {
			return "", err
		} else if encoded != "" {
			return encoded, nil
		}
	}

	return "", nil

}

// converts an iCalendar component into its string representation
func Marshal(target interface{}) (string, error) {

	// don't do anything with invalid interfaces
	v := reflect.ValueOf(target)
	if isInvalidOrEmptyValue(v) {
		return "", utils.NewError(Marshal, "unable to marshal empty or invalid values", target, nil)
	}

	if encoded, err := encode(v, propertyEncoder, objectEncoder, stringEncoder); err != nil {
		return "", err
	} else if encoded == "" {
		return "", utils.NewError(Marshal, "unable to encode interface, all methods exhausted", v.Interface(), nil)
	} else {
		return encoded, nil
	}

}
