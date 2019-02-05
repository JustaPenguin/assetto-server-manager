package servermanager

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/fatih/camelcase"
	"github.com/sirupsen/logrus"
)

const (
	formTypeTagName = "input"
	formOptsTagName = "formopts"
	formShowTagName = "show"
)

func NewForm(i interface{}, dropdownOpts map[string][]string, visibility string) *Form {
	return &Form{
		data:            i,
		dropdownOptions: dropdownOpts,
		visibility:      visibility,
	}
}

type Form struct {
	// the data on which the form is based
	data interface{}

	dropdownOptions map[string][]string

	// visibility is used to filter out fields by their 'show' tag.
	visibility string
}

func (f Form) Submit(r *http.Request) error {
	if reflect.ValueOf(f.data).Kind() != reflect.Ptr {
		panic("form data must be a pointer to a type")
	}

	if err := r.ParseForm(); err != nil {
		return err
	}

	val := reflect.ValueOf(f.data).Elem()

	for name, vals := range r.Form {
		f := val.FieldByName(name)

		if f.IsValid() && f.CanSet() {
			switch f.Kind() {
			case reflect.String:
				f.SetString(strings.Join(vals, ";"))
			case reflect.Int:
				if vals[0] == "on" {
					f.SetInt(1)
				} else {
					i, err := strconv.ParseInt(vals[0], 10, 0)

					if err != nil {
						return err
					}

					f.SetInt(i)
				}
			default:
				panic("form submit - unknown type")
			}
		}
	}

	return nil
}

func (f Form) Fields() []FormElement {
	if reflect.ValueOf(f.data).Kind() != reflect.Ptr {
		panic("form data must be a pointer to a type")
	}

	val := reflect.ValueOf(f.data)
	t := reflect.TypeOf(f.data)

	opts := f.buildOpts(val.Elem(), t.Elem(), "")

	return opts
}

type FormHeader struct {
	Name string
}

func (fh FormHeader) HTML() template.HTML {
	return template.HTML("<h2>" + fh.Name + "</h2>")
}

type FormElement interface {
	HTML() template.HTML
}

func (f Form) buildOpts(val reflect.Value, t reflect.Type, parentName string) []FormElement {
	var opts []FormElement

	for i := 0; i < t.NumField(); i++ {
		typeField := t.Field(i)
		valField := val.Field(i)

		switch valField.Kind() {
		case reflect.Struct:
			opts = append(opts, f.buildOpts(valField, typeField.Type, fmt.Sprintf("%s%s.", parentName, typeField.Name))...)
		case reflect.Map:
			for _, k := range valField.MapKeys() {
				elem := valField.MapIndex(k)

				opts = append(opts, FormHeader{Name: k.String()})
				opts = append(opts, f.buildOpts(elem, elem.Type(), fmt.Sprintf("%s%s[%s].", parentName, typeField.Name, k.String()))...)
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < valField.Len(); i++ {
				elem := valField.Index(i)

				opts = append(opts, f.buildOpts(elem, elem.Type(), fmt.Sprintf("%s%s[%d].", parentName, typeField.Name, i))...)
			}

		default:
			formShow := typeField.Tag.Get(formShowTagName)

			// check to see if we should be showing this tag
			if f.visibility != "" && formShow != f.visibility {
				continue
			}

			// formType can be e.g. dropdown
			formType := typeField.Tag.Get(formTypeTagName)

			if formType == "" {
				formType = typeField.Type.String()
			}

			formOpt := FormOption{
				Name:     strings.Join(camelcase.Split(typeField.Name), " "),
				Key:      typeField.Name,
				Value:    valField.Interface(),
				HelpText: typeField.Tag.Get("help"),
				Type:     formType,
				Opts:     make(map[string]bool),
			}

			if formType == "dropdown" || formType == "multiSelect" {
				optsKey := typeField.Tag.Get(formOptsTagName)

				if options, ok := f.dropdownOptions[optsKey]; ok {
					for _, o := range options {
						formOpt.Opts[o] = strings.Contains(formOpt.Value.(string), o)
					}
				} else {
					logrus.Warnf("dropdown opts for field: %s specified, but none found", typeField.Name)
				}
			} else if formType == "int" || formType == "number" {
				formOpt.Min, formOpt.Max = typeField.Tag.Get("min"), typeField.Tag.Get("max")
			}

			opts = append(opts, formOpt)
		}
	}

	return opts
}

type FormOption struct {
	Name     string
	Key      string
	Type     string
	HelpText string
	Value    interface{}
	Min, Max string

	Opts map[string]bool
}

func (f FormOption) render(templ string) (template.HTML, error) {
	t, err := template.New(f.Name).Parse(templ)

	if err != nil {
		return "", err
	}

	out := new(bytes.Buffer)

	err = t.Execute(out, f)

	if err != nil {
		return "", err
	}

	return template.HTML(out.String()), nil
}

func (f FormOption) renderDropdown() template.HTML {
	const dropdownTemplate = `
		<div class="form-group">
			<label>
				{{ .Name }}
				<select {{ if eq .Type "multiSelect" }} multiple size="8" {{ end }} class="form-control" name="{{ .Key }}" id="{{ .Key }}">
					{{ range $opt, $selected := .Opts }}
						<option {{ if $selected }} selected {{ end }} value="{{ $opt }}">{{ $opt }}</option>
					{{ end }}
				</select>

				<small>{{ .HelpText }}</small>
			</label>
		</div>
	`

	tmpl, err := f.render(dropdownTemplate)

	if err != nil {
		return template.HTML(fmt.Sprintf("err: %s", err))
	}

	return tmpl
}

func (f FormOption) renderCheckbox() template.HTML {
	const checkboxTemplate = `
		<div class="form-group">
			<label for="{{ .Key }}">{{ .Name }}</label>
			<input type="checkbox" id="{{ .Key }}" name="{{ .Key }}" {{ if eq .Value 1 }}checked="checked"{{ end }}><br>

			<small>{{ .HelpText }}</small>
		</div>
	`

	tmpl, err := f.render(checkboxTemplate)

	if err != nil {
		return template.HTML(fmt.Sprintf("err: %s", err))
	}

	return tmpl
}

func (f FormOption) renderTextInput() template.HTML {
	const inputTextTemplate = `
		<div class="form-group">
			<label for="{{ .Key }}">{{ .Name }}</label>
			<input type="{{ if eq .Type "password" }}password{{ else }}text{{ end }}" id="{{ .Key }}" name="{{ .Key }}" class="form-control" value="{{ .Value }}">

			<small>{{ .HelpText }}</small>
		</div>
	`

	tmpl, err := f.render(inputTextTemplate)

	if err != nil {
		return template.HTML(fmt.Sprintf("err: %s", err))
	}

	return tmpl
}

func (f FormOption) renderNumberInput() template.HTML {
	const numberInputTemplate = `
		<div class="form-group">
			<label for="{{ .Key }}">{{ .Name }}</label>
			<input 
				type="number" 
				id="{{ .Key }}" 
				name="{{ .Key }}" 
				class="form-control" 
				value="{{ .Value }}"
				{{ with .Min }}min="{{ . }}"{{ end }}
				{{ with .Max }}min="{{ . }}"{{ end }}
				step="1"
			>

			<small>{{ .HelpText }}</small>
		</div>
	`

	tmpl, err := f.render(numberInputTemplate)

	if err != nil {
		return template.HTML(fmt.Sprintf("err: %s", err))
	}

	return tmpl
}

func (f FormOption) HTML() template.HTML {
	switch f.Type {
	case "dropdown", "multiSelect":
		return f.renderDropdown()
	case "checkbox":
		return f.renderCheckbox()
	case "int":
		if f.Value == nil {
			f.Value = 0
		}

		return f.renderNumberInput()
	case "string", "password":
		return f.renderTextInput()
	default:
		logrus.Errorf("Unknown type: %s", f.Type)
		return ""
	}
}
