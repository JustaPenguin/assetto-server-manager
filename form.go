package servermanager

import (
	"fmt"
	"html/template"
	"reflect"
	"strings"

	"github.com/fatih/camelcase"
	"github.com/sirupsen/logrus"
)

type FormOption struct {
	Name     string
	Key      string
	Type     string
	HelpText string
	Value    interface{}
	Min, Max int
}

func (f FormOption) HTML() template.HTML {
	if f.Type == "checkbox" {
		return template.HTML(fmt.Sprintf(`<label>%s <input type="checkbox" name="%s" value="%d"></label><br><small>%s</small><br><br>`, f.Name, f.Key, f.Value, f.HelpText))
	} else if f.Type == "int" {
		if f.Value == nil {
			f.Value = 0
		}

		return template.HTML(fmt.Sprintf(`<label>%s <input type="number" name="%s" value="%d"></label><br><small>%s</small><br><br>`, f.Name, f.Key, f.Value, f.HelpText))
	} else if f.Type == "string" {
		if f.Value == nil {
			f.Value = ""
		}

		return template.HTML(fmt.Sprintf(`<label>%s <input type="text" name="%s" value="%s"></label><br><small>%s</small><br><br>`, f.Name, f.Key, f.Value, f.HelpText))
	} else {
		logrus.Errorf("Unknown type: %s", f.Type)
		return ""
	}
}

func NewForm(i interface{}) *Form {
	return &Form{
		data: i,
	}
}

type Form struct {
	data interface{}
}

func (f Form) Fields() []FormOption {
	t := reflect.TypeOf(f.data)
	val := reflect.ValueOf(f.data)

	var opts []FormOption

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		ft := field.Tag.Get("formtype")

		if ft == "" {
			ft = field.Type.String()
		}

		opts = append(opts, FormOption{
			Name:     strings.Join(camelcase.Split(field.Name), " "),
			Key:      field.Name,
			Value:    val.Field(i).Interface(),
			HelpText: field.Tag.Get("help"),
			Type:     ft,
		})
	}

	return opts
}
