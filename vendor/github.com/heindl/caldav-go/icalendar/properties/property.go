package properties

import (
	"fmt"
	"github.com/heindl/caldav-go/utils"
	"log"
	"reflect"
	"strings"
)

var _ = log.Print

var propNameSanitizer = strings.NewReplacer(
	"_", "-",
	":", "\\:",
)

var propValueSanitizer = strings.NewReplacer(
	"\"", "'",
	"\\", "\\\\",
	"\n", "\\n",
)

var propNameDesanitizer = strings.NewReplacer(
	"-", "_",
	"\\:", ":",
)

var propValueDesanitizer = strings.NewReplacer(
	"'", "\"",
	"\\\\", "\\",
	"\\n", "\n",
)

type Property struct {
	Name                PropertyName
	Value, DefaultValue string
	Params              Params
	OmitEmpty, Required bool
}

func (p *Property) HasNameAndValue() bool {
	return p.Name != "" && p.Value != ""
}

func (p *Property) Merge(override *Property) {
	if override.Name != "" {
		p.Name = override.Name
	}
	if override.Value != "" {
		p.Value = override.Value
	}
	if override.Params != nil {
		p.Params = override.Params
	}
}

func PropertyFromStructField(fs reflect.StructField) (p *Property) {

	ftag := fs.Tag.Get("ical")
	if fs.PkgPath != "" || ftag == "-" {
		return
	}

	p = new(Property)

	// parse the field tag
	if ftag != "" {
		tags := strings.Split(ftag, ",")
		p.Name = PropertyName(tags[0])
		if len(tags) > 1 {
			if tags[1] == "omitempty" {
				p.OmitEmpty = true
			} else if tags[1] == "required" {
				p.Required = true
			} else {
				p.DefaultValue = tags[1]
			}
		}
	}

	// make sure we have a name
	if p.Name == "" {
		p.Name = PropertyName(fs.Name)
	}

	p.Name = PropertyName(strings.ToUpper(string(p.Name)))

	return

}

func MarshalProperty(p *Property) string {
	name := strings.ToUpper(propNameSanitizer.Replace(string(p.Name)))
	value := propValueSanitizer.Replace(p.Value)
	keys := []string{name}
	for name, value := range p.Params {
		name = ParameterName(strings.ToUpper(propNameSanitizer.Replace(string(name))))
		value = propValueSanitizer.Replace(value)
		if strings.ContainsAny(value, " :") {
			keys = append(keys, fmt.Sprintf("%s=\"%s\"", name, value))
		} else {
			keys = append(keys, fmt.Sprintf("%s=%s", name, value))
		}
	}
	name = strings.Join(keys, ";")
	return fmt.Sprintf("%s:%s", name, value)
}

func PropertyFromInterface(target interface{}) (p *Property, err error) {

	var ierr error
	if va, ok := target.(CanValidateValue); ok {
		if ierr = va.ValidateICalValue(); ierr != nil {
			err = utils.NewError(PropertyFromInterface, "interface failed validation", target, ierr)
			return
		}
	}

	p = new(Property)

	if enc, ok := target.(CanEncodeName); ok {
		if p.Name, ierr = enc.EncodeICalName(); ierr != nil {
			err = utils.NewError(PropertyFromInterface, "interface failed name encoding", target, ierr)
			return
		}
	}

	if enc, ok := target.(CanEncodeParams); ok {
		if p.Params, ierr = enc.EncodeICalParams(); ierr != nil {
			err = utils.NewError(PropertyFromInterface, "interface failed params encoding", target, ierr)
			return
		}
	}

	if enc, ok := target.(CanEncodeValue); ok {
		if p.Value, ierr = enc.EncodeICalValue(); ierr != nil {
			err = utils.NewError(PropertyFromInterface, "interface failed value encoding", target, ierr)
			return
		}
	}

	return

}

func UnmarshalProperty(line string) *Property {
	nvp := strings.SplitN(line, ":", 2)
	prop := new(Property)
	if len(nvp) > 1 {
		prop.Value = strings.TrimSpace(nvp[1])
	}
	npp := strings.Split(nvp[0], ";")
	if len(npp) > 1 {
		prop.Params = make(map[ParameterName]string, 0)
		for i := 1; i < len(npp); i++ {
			var key, value string
			kvp := strings.Split(npp[i], "=")
			key = strings.TrimSpace(kvp[0])
			key = propNameDesanitizer.Replace(key)
			if len(kvp) > 1 {
				value = strings.TrimSpace(kvp[1])
				value = propValueDesanitizer.Replace(value)
				value = strings.Trim(value, "\"")
			}
			prop.Params[ParameterName(key)] = value
		}
	}
	prop.Name = PropertyName(strings.TrimSpace(npp[0]))
	prop.Name = PropertyName(propNameDesanitizer.Replace(string(prop.Name)))
	prop.Value = propValueDesanitizer.Replace(prop.Value)
	return prop
}

func NewProperty(name, value string) *Property {
	return &Property{Name: PropertyName(name), Value: value}
}
