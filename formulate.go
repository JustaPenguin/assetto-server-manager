package servermanager

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/cj123/formulate"
	"github.com/cj123/formulate/decorators"
	"golang.org/x/net/html"
)

// EncodeFormData uses formulate to build template.HTML from a struct
func EncodeFormData(data interface{}, r *http.Request) (template.HTML, error) {
	buf := new(bytes.Buffer)

	enc := formulate.NewEncoder(buf, &decorator{})
	enc.SetFormat(true)
	enc.AddShowCondition("premium", Premium)
	enc.AddShowCondition("open", func() bool {
		if !IsHosted {
			return true
		}

		account := AccountFromRequest(r)

		return account.Name == adminUserName
	})
	enc.AddShowCondition("read", func() bool {
		return ReadAccess(r)()
	})
	enc.AddShowCondition("write", func() bool {
		return WriteAccess(r)()
	})
	enc.AddShowCondition("delete", func() bool {
		return DeleteAccess(r)()
	})
	enc.AddShowCondition("admin", func() bool {
		return AdminAccess(r)()
	})

	if err := enc.Encode(data); err != nil {
		return "", err
	}

	return template.HTML(buf.String()), nil
}

func DecodeFormData(out interface{}, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	return formulate.NewDecoder(r.Form).Decode(out)
}

type FormHeading string

func (f FormHeading) BuildFormElement(_ string, parent *html.Node, field formulate.StructField, _ formulate.Decorator) error {
	// scrap the label
	parent = parent.Parent
	parent.RemoveChild(parent.FirstChild)

	div := &html.Node{
		Type: html.ElementNode,
		Data: "div",
	}

	formulate.AppendClass(div, "col-12")

	div.AppendChild(&html.Node{
		Type: html.ElementNode,
		Data: "hr",
		Attr: []html.Attribute{
			{
				Key: "class",
				Val: "mt-5",
			},
		},
	})

	h2 := &html.Node{
		Type: html.ElementNode,
		Data: "h3",
	}

	h2.AppendChild(&html.Node{
		Type: html.TextNode,
		Data: field.GetName(),
	})

	div.AppendChild(h2)

	parent.AppendChild(div)

	return nil
}

type decorator struct {
	decorators.BootstrapDecorator

	numFieldSets int
}

// HelpText trusts the fields helptext to be safe html.
func (d *decorator) HelpText(n *html.Node, field formulate.StructField) {
	n.Data = "div"

	helpText, err := html.Parse(strings.NewReader(field.GetHelpText()))

	if err != nil {
		return
	}

	n.FirstChild = helpText

	d.BootstrapDecorator.HelpText(n, field)
}

func (d *decorator) Label(n *html.Node, field formulate.StructField) {
	d.col3(n)
}

func (d *decorator) FieldWrapper(n *html.Node, field formulate.StructField) {
	d.col9(n)
}

func (d *decorator) col3(n *html.Node) {
	formulate.AppendClass(n, "col-sm-3 col-12")
}

func (d *decorator) col9(n *html.Node) {
	formulate.AppendClass(n, "col-sm-9 col-12")
}

func (d *decorator) TextField(n *html.Node, field formulate.StructField) {
	d.BootstrapDecorator.TextField(n, field)

	for _, attr := range n.Attr {
		if attr.Key == "type" && attr.Val == "password" {
			n.Attr = append(n.Attr, html.Attribute{
				Key: "autocomplete",
				Val: "new-password",
			})
		}
	}
}

func (d *decorator) TextareaField(n *html.Node, field formulate.StructField) {
	d.BootstrapDecorator.TextareaField(n, field)

	n.Attr = append(n.Attr, html.Attribute{
		Key: "rows",
		Val: "15",
	})
}

func (d *decorator) Fieldset(n *html.Node, field formulate.StructField) {
	if d.numFieldSets == 0 {
		d.BootstrapDecorator.Fieldset(n, field)
	} else {
		parent := n.Parent
		n.Parent.RemoveChild(n)

		card := &html.Node{
			Type: html.ElementNode,
			Data: "div",
		}
		formulate.AppendClass(card, "card", "mt-2", "mb-4")
		parent.AppendChild(card)

		heading := &html.Node{
			Type: html.ElementNode,
			Data: "div",
		}
		formulate.AppendClass(heading, "card-header")

		heading.AppendChild(&html.Node{
			Type: html.TextNode,
			Data: field.GetName(),
		})

		n.Data = "div"

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			n.RemoveChild(c)
		}

		formulate.AppendClass(n, "card-body")

		card.AppendChild(heading)
		card.AppendChild(n)
	}

	d.numFieldSets++
}

type boolString string

func (b boolString) BuildFormElement(key string, parent *html.Node, field formulate.StructField, decorator formulate.Decorator) error {
	n := &html.Node{
		Type: html.ElementNode,
		Data: "input",
		Attr: []html.Attribute{
			{
				Key: "type",
				Val: "checkbox",
			},
			{
				Key: "name",
				Val: key,
			},
			{
				Key: "id",
				Val: key,
			},
		},
	}

	checked := b == "true"

	if checked {
		n.Attr = append(n.Attr, html.Attribute{Key: "checked", Val: "checked"})
	}

	parent.AppendChild(n)
	decorator.CheckboxField(n, field)

	return nil
}

func (b boolString) DecodeFormValue(form url.Values, name string, values []string) (reflect.Value, error) {
	if len(values) == 0 {
		return reflect.Value{}, fmt.Errorf("servermanager: invalid value length, expected 1, got: %d", len(values))
	}

	if formValueAsInt(values[0]) == 1 {
		return reflect.ValueOf(boolString("true")), nil
	}

	return reflect.ValueOf(boolString("false")), nil
}
