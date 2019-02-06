package servermanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/mattn/go-zglob"
	"github.com/sirupsen/logrus"
)

// Renderer is the template engine.
type Renderer struct {
	templates map[string]*template.Template
	dir       string
	reload    bool
	mutex     sync.Mutex
}

func NewRenderer(dir string, reload bool) (*Renderer, error) {
	tr := &Renderer{
		templates: make(map[string]*template.Template),
		dir:       dir,
		reload:    reload,
	}

	err := tr.init()

	if err != nil {
		return nil, err
	}

	return tr, nil
}

// init loads template files into memory.
func (tr *Renderer) init() error {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()

	pages, err := zglob.Glob(filepath.Join(tr.dir, "pages", "**", "*.html"))

	if err != nil {
		return err
	}

	partials, err := zglob.Glob(filepath.Join(tr.dir, "partials", "*.html"))

	if err != nil {
		return err
	}

	funcs := sprig.FuncMap()
	funcs["ordinal"] = ordinal
	funcs["prettify"] = prettifyName
	funcs["carList"] = carList
	funcs["jsonEncode"] = jsonEncode
	funcs["varSplit"] = varSplit
	funcs["timeFormat"] = timeFormat
	funcs["dateFormat"] = dateFormat

	for _, page := range pages {
		var templateList []string
		templateList = append(templateList, filepath.Join(tr.dir, "layout", "base.html"))
		templateList = append(templateList, partials...)
		templateList = append(templateList, page)

		t, err := template.New(page).Funcs(funcs).ParseFiles(templateList...)

		if err != nil {
			return err
		}

		tr.templates[filepath.ToSlash(page)] = t
	}

	return nil
}

func timeFormat(t time.Time) string {
	return t.Format(time.Kitchen)
}

func dateFormat(t time.Time) string {
	return t.Format("02/01/2006")
}

func carList(cars string) string {
	var out []string

	split := strings.Split(cars, ";")

	for _, s := range split {
		out = append(out, prettifyName(s, true))
	}

	return strings.Join(out, ", ")
}

func varSplit(str string) []string {
	return strings.Split(str, ";")
}

var nameRegex = regexp.MustCompile(`^[A-Za-z]{0,5}[0-9]+`)

func prettifyName(s string, acronyms bool) string {
	parts := strings.Split(s, "_")

	if parts[0] == "ks" {
		parts = parts[1:]
	}

	for i := range parts {
		if (acronyms && len(parts[i]) <= 3) || (acronyms && nameRegex.MatchString(parts[i])) {
			parts[i] = strings.ToUpper(parts[i])
		} else {
			parts[i] = strings.Title(parts[i])
		}
	}

	return strings.Join(parts, " ")
}

func jsonEncode(v interface{}) template.JS {
	buf := new(bytes.Buffer)

	_ = json.NewEncoder(buf).Encode(v)

	return template.JS(buf.String())
}

// LoadTemplate reads a template from templates and renders it with data to the given io.Writer
func (tr *Renderer) LoadTemplate(w http.ResponseWriter, r *http.Request, view string, data map[string]interface{}) error {
	if tr.reload {
		// reload templates on every request if enabled, so
		// that we don't have to constantly restart the website
		err := tr.init()

		if err != nil {
			return err
		}
	}

	pageView := filepath.Join(tr.dir, "pages", view)

	t, ok := tr.templates[filepath.ToSlash(pageView)]

	if !ok {
		return fmt.Errorf("unable to find template: %s", pageView)
	}

	if data == nil {
		data = make(map[string]interface{})
	}

	session, err := getSession(r)

	if err != nil {
		return err
	}

	if flashes := session.Flashes(); len(flashes) > 0 {
		data["messages"] = flashes
	}

	err = session.Save(r, w)

	if err != nil {
		return err
	}

	data["server_status"] = AssettoProcess.IsRunning()

	return t.ExecuteTemplate(w, "base", data)
}

// MustLoadTemplate asserts that a LoadTemplate call must succeed or be dealt with via the http.ResponseWriter
func (tr *Renderer) MustLoadTemplate(w http.ResponseWriter, r *http.Request, view string, data map[string]interface{}) {
	err := tr.LoadTemplate(w, r, view, data)

	if err != nil {
		logrus.Errorf("Unable to load template: %s, err: %s", view, err)
		http.Error(w, "unable to load template", http.StatusInternalServerError)
		return
	}
}

func (tr *Renderer) LoadPartial(w http.ResponseWriter, partial string, data map[string]interface{}) error {
	path := filepath.Join(tr.dir, "pages", partial)

	t, err := template.New(partial).ParseFiles(path)

	if err != nil {
		return err
	}

	return t.Execute(w, data)
}

func (tr *Renderer) MustLoadPartial(w http.ResponseWriter, partial string, data map[string]interface{}) {
	err := tr.LoadPartial(w, partial, data)

	if err != nil {
		logrus.Errorf("Unable to load partial: %s, err: %s", partial, err)
		http.Error(w, "unable to load partial", http.StatusInternalServerError)
		return
	}
}

func ordinal(num int64) string {
	suffix := "th"
	switch num {
	case 1, 21, 31:
		suffix = "st"
	case 2, 22:
		suffix = "nd"
	case 3, 23:
		suffix = "rd"
	}

	return suffix
}
