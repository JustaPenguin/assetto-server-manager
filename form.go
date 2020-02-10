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

func NewForm(i interface{}, dropdownOpts map[string][]string, visibility string, forceShowAllOptions bool) *Form {
	return &Form{
		data:                i,
		dropdownOptions:     dropdownOpts,
		visibility:          visibility,
		forceShowAllOptions: forceShowAllOptions,
	}
}

type Form struct {
	// the data on which the form is based
	data interface{}

	dropdownOptions map[string][]string

	// visibility is used to filter out fields by their 'show' tag.
	visibility string

	// force all options to show
	forceShowAllOptions bool
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
		err := f.assignFieldValues(val, name, vals)

		if err != nil {
			return err
		}
	}

	return nil
}

func (f Form) assignFieldValues(val reflect.Value, name string, vals []string) error {
	parts := strings.Split(name, ".")
	field := val.FieldByName(parts[0])

	if field.IsValid() && field.CanSet() {
		switch field.Kind() {
		case reflect.Struct:
			if len(parts) > 1 {
				err := f.assignFieldValues(field, strings.Join(parts[1:], "."), vals)

				if err != nil {
					return err
				}
			}
		case reflect.String:
			field.SetString(strings.Join(vals, ";"))
		case reflect.Int:
			if vals[0] == "on" {
				field.SetInt(1)
			} else {
				field.SetInt(int64(formValueAsInt(vals[0])))
			}
		case reflect.Float64:
			f, err := strconv.ParseFloat(vals[0], 64)

			if err != nil {
				return err
			}

			field.SetFloat(f)
		case reflect.Bool:
			if vals[0] == "on" {
				field.SetBool(true)
			} else {
				field.SetBool(formValueAsInt(vals[0]) == 1)
			}
		default:
			panic("form submit - unknown type: " + field.Kind().String())
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

	opts := f.buildOpts(val.Elem(), t.Elem(), "", 0)

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

func (f Form) buildName(name string) string {
	return strings.Replace(strings.Join(camelcase.Split(name), " "), "ACSRAPI", "ACSR API", 1)
}

func (f Form) buildOpts(val reflect.Value, t reflect.Type, parentName string, recursionLevel int) []FormElement {
	var opts []FormElement

	for i := 0; i < t.NumField(); i++ {
		typeField := t.Field(i)
		valField := val.Field(i)

		formShow := typeField.Tag.Get(formShowTagName)

		// check to see if we should be showing this tag
		if formShow == "-" || (formShow != "open" && f.visibility != "" && formShow != f.visibility) || formShow == "premium" && !Premium() {
			continue
		}

		switch valField.Kind() {
		case reflect.Struct:
			if recursionLevel == 0 {
				opts = append(opts, formCardOpen{name: f.buildName(typeField.Name)})
			} else {
				opts = append(opts, FormOption{Name: f.buildName(typeField.Name), Type: "heading", RecursionLevel: recursionLevel})
			}

			opts = append(opts, f.buildOpts(valField, typeField.Type, fmt.Sprintf("%s%s.", parentName, typeField.Name), recursionLevel+1)...)

			if recursionLevel == 0 {
				opts = append(opts, formCardClose{})
			}
		case reflect.Map:
			for _, k := range valField.MapKeys() {
				elem := valField.MapIndex(k)

				opts = append(opts, FormHeader{Name: k.String()})
				opts = append(opts, f.buildOpts(elem, elem.Type(), fmt.Sprintf("%s%s[%s].", parentName, typeField.Name, k.String()), recursionLevel+1)...)
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < valField.Len(); i++ {
				elem := valField.Index(i)

				opts = append(opts, f.buildOpts(elem, elem.Type(), fmt.Sprintf("%s%s[%d].", parentName, typeField.Name, i), recursionLevel+1)...)
			}

		default:
			// formType can be e.g. dropdown
			formType := typeField.Tag.Get(formTypeTagName)

			if formType == "" {
				formType = typeField.Type.String()
			}

			formName := f.buildName(typeField.Name)

			formOpt := FormOption{
				Name:           formName,
				Key:            parentName + typeField.Name,
				Value:          valField.Interface(),
				HelpText:       template.HTML(typeField.Tag.Get("help")),
				Type:           formType,
				Opts:           make(map[string]bool),
				Hidden:         formShow == "open" && IsHosted && !f.forceShowAllOptions,
				RecursionLevel: recursionLevel,
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
	Name           string
	Key            string
	Type           string
	HelpText       template.HTML
	Value          interface{}
	Min, Max       string
	Hidden         bool
	RecursionLevel int

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
		{{ if not .Hidden }}
			<div class="form-group row">
				<label for="{{ .Key }}" class="col-sm-3 col-form-label">
					{{ .Name }}
				</label>

            	<div class="col-sm-9">
					<select {{ if eq .Type "multiSelect" }} multiple {{ end }} class="form-control" name="{{ .Key }}" id="{{ .Key }}">
						{{ range $opt, $selected := .Opts }}
							<option {{ if $selected }} selected {{ end }} value="{{ $opt }}">{{ $opt }}</option>
						{{ end }}
					</select>

					<small>{{ .HelpText }}</small>
				</div>
			</div>
		{{ else }}
			{{ range $opt, $selected := .Opts }}
				{{ if $selected }}
					<input type="hidden" name="{{ .Key }}" value="{{ $opt }}">
				{{ end }}
			{{ end }}
		{{ end }}
	`

	tmpl, err := f.render(dropdownTemplate)

	if err != nil {
		return template.HTML(fmt.Sprintf("err: %s", err))
	}

	return tmpl
}

func (f FormOption) renderCheckbox() template.HTML {
	if b, ok := f.Value.(bool); ok {
		if b {
			f.Value = 1
		} else {
			f.Value = 0
		}
	}

	const checkboxTemplate = `
		{{ if not .Hidden }}
			<div class="form-group row">
				<label for="{{ .Key }}" class="col-sm-3 col-form-label">{{ .Name }}</label>


            	<div class="col-sm-9">
					<input type="checkbox" id="{{ .Key }}" name="{{ .Key }}" {{ if eq .Value 1 }}checked="checked"{{ end }}><br>

					<small>{{ .HelpText }}</small>
				</div>
			</div>
		{{ else }}
			<input type="hidden" id="{{ .Key }}" name="{{ .Key }}" {{ if eq .Value 1 }}value="1"{{ else }}value="0"{{ end }}>
		{{ end }}
	`

	tmpl, err := f.render(checkboxTemplate)

	if err != nil {
		return template.HTML(fmt.Sprintf("err: %s", err))
	}

	return tmpl
}

func (f FormOption) renderTextInput() template.HTML {
	const inputTextTemplate = `
		{{ if not .Hidden }}
			<div class="form-group row">
				<label for="{{ .Key }}" class="col-sm-3 col-form-label">{{ .Name }}</label>

				<div class="col-sm-9">
					<input type="{{ if eq .Type "password" }}password{{ else }}text{{ end }}" id="{{ .Key }}" name="{{ .Key }}" class="form-control" value="{{ .Value }}">

					<small>{{ .HelpText }}</small>
				</div>
			</div>
		{{ else }}
			<input type="hidden" id="{{ .Key }}" name="{{ .Key }}" value="{{ .Value }}">
		{{ end }}
	`

	tmpl, err := f.render(inputTextTemplate)

	if err != nil {
		return template.HTML(fmt.Sprintf("err: %s", err))
	}

	return tmpl
}

func (f FormOption) renderTextarea() template.HTML {
	const textareaTemplate = `
		<div class="form-group row" {{ if .Hidden }}style="display: none;"{{ end }}>
			<label for="{{ .Key }}" class="col-sm-3 col-form-label">{{ .Name }}</label>

			<div class="col-sm-9">
				<textarea id="{{ .Key }}" name="{{ .Key }}" class="form-control text-monospace" rows="15">{{ .Value }}</textarea>

				<small>{{ .HelpText }}</small>
			</div>
		</div>
	`

	tmpl, err := f.render(textareaTemplate)

	if err != nil {
		return template.HTML(fmt.Sprintf("err: %s", err))
	}

	return tmpl
}

func (f FormOption) renderNumberInput() template.HTML {
	const numberInputTemplate = `
		{{ if not .Hidden }}
			<div class="form-group row">
				<label for="{{ .Key }}" class="col-sm-3 col-form-label">{{ .Name }}</label>

				<div class="col-sm-9">
					<input 
						type="number" 
						id="{{ .Key }}" 
						name="{{ .Key }}" 
						class="form-control" 
						value="{{ .Value }}"
						{{ with .Min }}min="{{ . }}"{{ end }}
						{{ with .Max }}min="{{ . }}"{{ end }}
						step="any"
					>

					<small>{{ .HelpText }}</small>
				</div>
			</div>
		{{ else }}
			<input type="hidden" id="{{ .Key }}" name="{{ .Key }}" value="{{ .Value }}">
		{{ end }}
	`

	tmpl, err := f.render(numberInputTemplate)

	if err != nil {
		return template.HTML(fmt.Sprintf("err: %s", err))
	}

	return tmpl
}

type FormHeading string

func (f FormOption) renderHeading() template.HTML {
	if f.Hidden {
		return ""
	}

	headerType := "h2"
	headingTemplate := ""

	switch f.RecursionLevel {
	case 0:
		headingTemplate = `<hr class="mt-5">`
	case 1:
		headerType = "h3"
	case 2:
		headerType = "h4"
	case 4:
		headerType = "h5"
	}

	headingTemplate += fmt.Sprintf(`<%s class="mt-4 mb-4">{{ .Name }}</%s>`, headerType, headerType)

	tmpl, err := f.render(headingTemplate)

	if err != nil {
		return template.HTML(fmt.Sprintf("err: %s", err))
	}

	return tmpl
}

func (f FormOption) HTML() template.HTML {
	switch f.Type {
	case "dropdown", "multiSelect":
		return f.renderDropdown()
	case "checkbox", "bool":
		return f.renderCheckbox()
	case "int", "float64":
		if f.Value == nil {
			f.Value = 0
		}

		return f.renderNumberInput()

	case "textarea", reflect.TypeOf(NewLineSeparatedList("")).String():
		return f.renderTextarea()
	case "string", "password":
		return f.renderTextInput()
	case "heading":
		return f.renderHeading()
	default:
		logrus.Errorf("Unknown type: %s", f.Type)
		return ""
	}
}

type formCardOpen struct {
	name string
}

func (f formCardOpen) HTML() template.HTML {
	return template.HTML(`
		<div class="card mt-3 mb-3">
			<div class="card-header">
				<strong>` + f.name + `</strong>
			</div>

			<div class="card-body">
`)
}

type formCardClose struct{}

func (f formCardClose) HTML() template.HTML {
	return `</div></div>`
}
