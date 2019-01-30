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
)

func NewForm(i interface{}, dropdownOpts map[string][]string) *Form {
	return &Form{
		data:            i,
		dropdownOptions: dropdownOpts,
	}
}

type Form struct {
	// the data on which the form is based
	data interface{}

	// dropdownOptions is accessed when forms specify a type 'dropdown' with formopts.
	// the formopts value becomes the key in this map.
	dropdownOptions map[string][]string
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

func (f Form) Fields() []FormOption {

	if reflect.ValueOf(f.data).Kind() != reflect.Ptr {
		panic("form data must be a pointer to a type")
	}
	val := reflect.ValueOf(f.data).Elem()

	t := reflect.TypeOf(f.data).Elem()

	var opts []FormOption

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// formType can be e.g. dropdown
		formType := field.Tag.Get(formTypeTagName)

		if formType == "" {
			formType = field.Type.String()
		}

		formOpt := FormOption{
			Name:     strings.Join(camelcase.Split(field.Name), " "),
			Key:      field.Name,
			Value:    val.Field(i).Interface(),
			HelpText: field.Tag.Get("help"),
			Type:     formType,
			Opts:     make(map[string]bool),
		}

		if formType == "dropdown" || formType == "multiSelect" {
			optsKey := field.Tag.Get(formOptsTagName)

			if options, ok := f.dropdownOptions[optsKey]; ok {
				for _, o := range options {
					formOpt.Opts[o] = strings.Contains(formOpt.Value.(string), o)
				}
			} else {
				logrus.Warnf("dropdown opts for field: %s specified, but none found", field.Name)
			}
		} else if formType == "int" || formType == "number" {
			formOpt.Min, formOpt.Max = field.Tag.Get("min"), field.Tag.Get("max")
		}

		opts = append(opts, formOpt)
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
				<select {{ if eq .Type "multiSelect" }} multiple {{ end }} class="form-control" name="{{ .Key }}">
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
