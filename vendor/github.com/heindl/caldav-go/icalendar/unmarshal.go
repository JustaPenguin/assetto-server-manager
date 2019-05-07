package icalendar

import (
	"fmt"
	"github.com/heindl/caldav-go/icalendar/properties"
	"github.com/heindl/caldav-go/utils"
	"log"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var _ = log.Print
var splitter = regexp.MustCompile("\r?\n")

type token struct {
	name       string
	components map[string][]*token
	properties map[properties.PropertyName][]*properties.Property
}

func tokenize(encoded string) (*token, error) {
	if encoded = strings.TrimSpace(encoded); encoded == "" {
		return nil, utils.NewError(tokenize, "no content to tokenize", encoded, nil)
	}
	return tokenizeSlice(splitter.Split(encoded, -1))
}

func tokenizeSlice(slice []string, name ...string) (*token, error) {

	tok := new(token)
	size := len(slice)

	if len(name) > 0 {
		tok.name = name[0]
	} else if size <= 0 {
		return nil, utils.NewError(tokenizeSlice, "token has no content", slice, nil)
	}

	tok.properties = make(map[properties.PropertyName][]*properties.Property, 0)
	tok.components = make(map[string][]*token, 0)

	for i := 0; i < size; i++ {



		// Handle iCalendar's space-indented line break format
		// See: https://www.ietf.org/rfc/rfc2445.txt section 4.1
		// "a long line can be split between any two characters by inserting a CRLF immediately followed by a single
		// linear white space character"
		line := slice[i]
		for ; i < size-1 && strings.HasPrefix(slice[i+1], " "); i++ {
			next := slice[i+1]
			line += next[1:len(next)]
		}
		prop := properties.UnmarshalProperty(line)

		if prop.Name.Equals("begin") {
			for j := i; j < size; j++ {
				end := strings.Replace(line, "BEGIN", "END", 1)
				if slice[j] == end {
					if component, err := tokenizeSlice(slice[i+1:j], prop.Value); err != nil {
						msg := fmt.Sprintf("unable to tokenize %s component", prop.Value)
						return nil, utils.NewError(tokenizeSlice, msg, slice, err)
					} else {
						existing, _ := tok.components[prop.Value]
						tok.components[prop.Value] = append(existing, component)
						i = j
						break
					}
				}
			}
		} else if existing, ok := tok.properties[prop.Name]; ok {
			tok.properties[prop.Name] = append(existing, prop)
		} else {
			tok.properties[prop.Name] = []*properties.Property{prop}
		}
	}

	return tok, nil
}

func hydrateInterface(v reflect.Value, prop *properties.Property) (bool, error) {

	// unable to decode into empty values
	if isInvalidOrEmptyValue(v) {
		return false, nil
	}

	var i = v.Interface()
	var hasValue = false

	// decode a value if possible
	if decoder, ok := i.(properties.CanDecodeValue); ok {
		if err := decoder.DecodeICalValue(prop.Value); err != nil {
			return false, utils.NewError(hydrateInterface, "error decoding property value", v, err)
		} else {
			hasValue = true
		}
	}

	// decode any params, if supported
	if len(prop.Params) > 0 {
		if decoder, ok := i.(properties.CanDecodeParams); ok {
			if err := decoder.DecodeICalParams(prop.Params); err != nil {
				return false, utils.NewError(hydrateInterface, "error decoding property parameters", v, err)
			}
		}
	}

	// finish with any validation
	if validator, ok := i.(properties.CanValidateValue); ok {
		if err := validator.ValidateICalValue(); err != nil {
			return false, utils.NewError(hydrateInterface, "error validating property value", v, err)
		}
	}

	return hasValue, nil

}

func hydrateLiteral(v reflect.Value, prop *properties.Property) (reflect.Value, error) {

	literal := dereferencePointerValue(v)

	switch literal.Kind() {
	case reflect.Bool:
		if i, err := strconv.ParseBool(prop.Value); err != nil {
			return literal, utils.NewError(hydrateLiteral, "unable to decode bool "+prop.Value, literal.Interface(), err)
		} else {
			literal.SetBool(i)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if i, err := strconv.ParseInt(prop.Value, 10, 64); err != nil {
			return literal, utils.NewError(hydrateLiteral, "unable to decode int "+prop.Value, literal.Interface(), err)
		} else {
			literal.SetInt(i)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if i, err := strconv.ParseUint(prop.Value, 10, 64); err != nil {
			return literal, utils.NewError(hydrateLiteral, "unable to decode uint "+prop.Value, literal.Interface(), err)
		} else {
			literal.SetUint(i)
		}
	case reflect.Float32, reflect.Float64:
		if i, err := strconv.ParseFloat(prop.Value, 64); err != nil {
			return literal, utils.NewError(hydrateLiteral, "unable to decode float "+prop.Value, literal.Interface(), err)
		} else {
			literal.SetFloat(i)
		}
	case reflect.String:
		literal.SetString(prop.Value)
	default:
		return literal, utils.NewError(hydrateLiteral, "unable to decode value as literal "+prop.Value, literal.Interface(), nil)
	}

	return literal, nil

}

func hydrateProperty(v reflect.Value, prop *properties.Property) error {

	// check to see if the interface handles it's own hydration
	if handled, err := hydrateInterface(v, prop); err != nil {
		return utils.NewError(hydrateProperty, "unable to hydrate interface", v, err)
	} else if handled {
		return nil // exit early if handled by the interface
	}

	// if we got here, we need to create a new instance to
	// set into the property.
	var vnew, varr = newValue(v)
	var vlit bool

	// check to see if the new value handles it's own hydration
	if handled, err := hydrateInterface(vnew, prop); err != nil {
		return utils.NewError(hydrateProperty, "unable to hydrate new interface value", vnew, err)
	} else if vlit = !handled; vlit {
		// if not, treat it as a literal
		if vnewlit, err := hydrateLiteral(vnew, prop); err != nil {
			return utils.NewError(hydrateProperty, "unable to hydrate new literal value", vnew, err)
		} else if _, err := hydrateInterface(vnewlit, prop); err != nil {
			return utils.NewError(hydrateProperty, "unable to hydrate new literal interface value", vnewlit, err)
		}
	}

	// now we can set the value
	vnewval := dereferencePointerValue(vnew)
	voldval := dereferencePointerValue(v)

	// make sure we can set the new value into the provided pointer

	if varr {
		// for arrays, append the new value into the array structure
		if !voldval.CanSet() {
			return utils.NewError(hydrateProperty, "unable to set array value", v, nil)
		} else {
			voldval.Set(reflect.Append(voldval, vnew))
			return nil
		}
	} else if vlit {
		// for literals, set the dereferenced value
		if !voldval.CanSet() {
			return utils.NewError(hydrateProperty, "unable to set literal value", v, nil)
		} else {
			voldval.Set(vnewval)
		}
	} else if !v.CanSet() {
		return utils.NewError(hydrateProperty, "unable to set pointer value", v, nil)
	} else {
		// everything else should be a pointer, set it directly
		v.Set(vnew)
	}

	return nil

}

func hydrateNestedComponent(v reflect.Value, component *token) error {

	// create a new object to hold the property value
	var vnew, varr = newValue(v)
	if err := hydrateComponent(vnew, component); err != nil {
		return utils.NewError(hydrateNestedComponent, "unable to decode component", component, err)
	}

	if varr {
		// for arrays, append the new value into the array structure
		voldval := dereferencePointerValue(v)
		if !voldval.CanSet() {
			return utils.NewError(hydrateNestedComponent, "unable to set array value", v, nil)
		} else {
			voldval.Set(reflect.Append(voldval, vnew))
		}
	} else if !v.CanSet() {
		return utils.NewError(hydrateNestedComponent, "unable to set pointer value", v, nil)
	} else {
		// everything else should be a pointer, set it directly
		v.Set(vnew)
	}

	return nil

}

func hydrateProperties(v reflect.Value, component *token) error {

	vdref := dereferencePointerValue(v)
	vtype := vdref.Type()
	vkind := vdref.Kind()

	if vkind != reflect.Struct {
		return utils.NewError(hydrateProperties, "unable to hydrate properties of non-struct", v, nil)
	}

	n := vtype.NumField()
	for i := 0; i < n; i++ {

		prop := properties.PropertyFromStructField(vtype.Field(i))
		if prop == nil {
			continue // skip if field is ignored
		}

		vfield := vdref.Field(i)

		// first try to hydrate property values
		if properties, ok := component.properties[prop.Name]; ok {
			for _, prop := range properties {
				if err := hydrateProperty(vfield, prop); err != nil {
					msg := fmt.Sprintf("unable to hydrate property %s", prop.Name)
					return utils.NewError(hydrateProperties, msg, v, err)
				}
			}
		}

		// then try to hydrate components
		vtemp, _ := newValue(vfield)
		if tag, err := extractTagFromValue(vtemp); err != nil {
			msg := fmt.Sprintf("unable to extract tag from property %s", prop.Name)
			return utils.NewError(hydrateProperties, msg, v, err)
		} else if components, ok := component.components[tag]; ok {
			for _, comp := range components {
				if err := hydrateNestedComponent(vfield, comp); err != nil {
					msg := fmt.Sprintf("unable to hydrate component %s", prop.Name)
					return utils.NewError(hydrateProperties, msg, v, err)
				}
			}
		}
	}

	return nil

}

func hydrateComponent(v reflect.Value, component *token) error {
	if tag, err := extractTagFromValue(v); err != nil {
		return utils.NewError(hydrateComponent, "error extracting tag from value", component, err)
	} else if tag != component.name {
		msg := fmt.Sprintf("expected %s and found %s", tag, component.name)
		return utils.NewError(hydrateComponent, msg, component, nil)
	} else if err := hydrateProperties(v, component); err != nil {
		return utils.NewError(hydrateComponent, "unable to hydrate properties", component, err)
	}
	return nil
}

func hydrateComponents(v reflect.Value, components []*token) error {
	vdref := dereferencePointerValue(v)
	for i, component := range components {
		velem := reflect.New(vdref.Type().Elem())
		if err := hydrateComponent(velem, component); err != nil {
			msg := fmt.Sprintf("unable to hydrate component %d", i)
			return utils.NewError(hydrateComponent, msg, component, err)
		} else {
			v.Set(reflect.Append(vdref, velem))
		}
	}
	return nil
}

func hydrateValue(v reflect.Value, component *token) error {

	if !v.IsValid() || v.Kind() != reflect.Ptr {
		return utils.NewError(hydrateValue, "unmarshal target must be a valid pointer", v, nil)
	}

	// handle any encodable properties
	if encoder, isprop := v.Interface().(properties.CanEncodeName); isprop {
		if name, err := encoder.EncodeICalName(); err != nil {
			return utils.NewError(hydrateValue, "unable to lookup property name", v, err)
		} else if properties, found := component.properties[name]; !found || len(properties) == 0 {
			return utils.NewError(hydrateValue, "no matching propery values found for "+string(name), v, nil)
		} else if len(properties) > 1 {
			return utils.NewError(hydrateValue, "more than one property value matches single property interface", v, nil)
		} else {
			return hydrateProperty(v, properties[0])
		}
	}

	// handle components
	vkind := dereferencePointerValue(v).Kind()
	if tag, err := extractTagFromValue(v); err != nil {
		return utils.NewError(hydrateValue, "unable to extract component tag", v, err)
	} else if components, found := component.components[tag]; !found || len(components) == 0 {
		msg := fmt.Sprintf("unable to find matching component for %s", tag)
		return utils.NewError(hydrateValue, msg, v, nil)
	} else if vkind == reflect.Array || vkind == reflect.Slice {
		return hydrateComponents(v, components)
	} else if len(components) > 1 {
		return utils.NewError(hydrateValue, "non-array interface provided but more than one component found!", v, nil)
	} else {
		return hydrateComponent(v, components[0])
	}

}

// decodes encoded icalendar data into a native interface
func Unmarshal(encoded string, into interface{}) error {
	if component, err := tokenize(encoded); err != nil {
		return utils.NewError(Unmarshal, "unable to tokenize encoded data", encoded, err)
	} else {
		return hydrateValue(reflect.ValueOf(into), component)
	}
}