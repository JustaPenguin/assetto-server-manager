package servermanager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/sprig"
	"github.com/mattn/go-zglob"
	"github.com/sirupsen/logrus"
)

// BuildTime is the time Server Manager was built at
var BuildTime string

type TemplateLoader interface {
	Init() error
	Templates(funcs template.FuncMap) (map[string]*template.Template, error)
}

func NewFilesystemTemplateLoader(dir string) TemplateLoader {
	return &filesystemTemplateLoader{
		dir: dir,
	}
}

type filesystemTemplateLoader struct {
	dir string

	pages, partials []string
}

func (fs *filesystemTemplateLoader) Init() error {
	var err error

	fs.pages, err = zglob.Glob(filepath.Join(fs.dir, "pages", "**", "*.html"))

	if err != nil {
		return err
	}

	fs.partials, err = zglob.Glob(filepath.Join(fs.dir, "partials", "**", "*.html"))

	if err != nil {
		return err
	}

	return nil
}

func (fs *filesystemTemplateLoader) Templates(funcs template.FuncMap) (map[string]*template.Template, error) {
	templates := make(map[string]*template.Template)

	for _, page := range fs.pages {
		var templateList []string
		templateList = append(templateList, filepath.Join(fs.dir, "layout", "base.html"))
		templateList = append(templateList, fs.partials...)
		templateList = append(templateList, page)

		t, err := template.New(page).Funcs(funcs).ParseFiles(templateList...)

		if err != nil {
			return nil, err
		}

		templates[strings.TrimPrefix(filepath.ToSlash(page), filepath.ToSlash(fs.dir)+"/pages/")] = t
	}

	return templates, nil
}

// Renderer is the template engine.
type Renderer struct {
	templates map[string]*template.Template

	loader TemplateLoader

	reload bool
	mutex  sync.Mutex
}

func NewRenderer(loader TemplateLoader, reload bool) (*Renderer, error) {
	tr := &Renderer{
		templates: make(map[string]*template.Template),
		loader:    loader,
		reload:    reload,
	}

	err := tr.init()

	if err != nil {
		return nil, err
	}

	return tr, nil
}

// dummy access func is used in place of read/write/admin funcs when initialising templates
func dummyAccessFunc() bool {
	return false
}

// init loads template files into memory.
func (tr *Renderer) init() error {
	tr.mutex.Lock()
	defer tr.mutex.Unlock()

	err := tr.loader.Init()

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
	funcs["timeZone"] = timeZone
	funcs["isBefore"] = isBefore
	funcs["trackInfo"] = trackInfo
	funcs["stripGeotagCrap"] = stripGeotagCrap
	funcs["ReadAccess"] = dummyAccessFunc
	funcs["WriteAccess"] = dummyAccessFunc
	funcs["DeleteAccess"] = dummyAccessFunc
	funcs["AdminAccess"] = dummyAccessFunc
	funcs["LoggedIn"] = dummyAccessFunc
	funcs["classColor"] = func(i int) string {
		return ChampionshipClassColors[i%len(ChampionshipClassColors)]
	}
	funcs["carSkinURL"] = carSkinURL
	funcs["dict"] = templateDict
	funcs["asset"] = NewAssetHelper("/", "", "", map[string]string{"cb": BuildTime}).GetURL
	funcs["SessionType"] = func(s string) SessionType { return SessionType(s) }
	funcs["Config"] = func() *Configuration { return config }
	funcs["Version"] = func() string { return BuildTime }

	tr.templates, err = tr.loader.Templates(funcs)

	if err != nil {
		return err
	}

	return nil
}

func templateDict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("invalid dict call")
	}
	dict := make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

func timeFormat(t time.Time) string {
	return t.Format(time.Kitchen)
}

func dateFormat(t time.Time) string {
	return t.Format("02/01/2006")
}

func timeZone(t time.Time) string {
	name, _ := t.Zone()

	return name
}

func isBefore(t time.Time) bool {
	return time.Now().Before(t)
}

func carList(cars interface{}) string {
	var split []string

	switch cars := cars.(type) {
	case string:
		split = strings.Split(cars, ";")
	case []string:
		split = cars
	default:
		panic("unknown type of cars: " + reflect.TypeOf(cars).String())
	}

	var out []string

	for _, s := range split {
		out = append(out, prettifyName(s, true))
	}

	sort.Strings(out)

	return strings.Join(out, ", ")
}

func varSplit(str string) []string {
	return strings.Split(str, ";")
}

var trackInfoCache = make(map[string]*TrackInfo)

func trackInfo(track, layout string) *TrackInfo {
	if t, ok := trackInfoCache[track+layout]; ok {
		return t
	}

	t, err := GetTrackInfo(track, layout)

	if err != nil {
		logrus.Errorf("Could not get track info for %s (%s), err: %s", track, layout, err)
		return nil
	}

	trackInfoCache[track+layout] = t

	return t
}

func stripGeotagCrap(tag string, north bool) string {
	re := regexp.MustCompile("[0-9]+")
	geoTags := re.FindAllString(tag, -1)

	if len(geoTags) == 2 {
		// "52.9452° N" format
		// @TODO +- some amount for the bbox

		return geoTags[0] + "." + geoTags[1]
	} else if len(geoTags) == 3 {
		// "50� 13' 57 N" format
		for _, thing := range geoTags {
			println(thing)
		}
	} else if len(geoTags) == 1 {
		// dunno, some crazy format, just return
		return geoTags[0]
	}

	// Geotags of "lost" - a hamlet in Scotland
	if north {
		return "57.2050"
	} else {
		return "-3.0774"
	}

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
			parts[i] = strings.Title(strings.ToLower(parts[i]))
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

	t, ok := tr.templates[filepath.ToSlash(view)]

	if !ok {
		return fmt.Errorf("unable to find template: %s", filepath.ToSlash(view))
	}

	if data == nil {
		data = make(map[string]interface{})
	}

	session := getSession(r)

	if flashes := session.Flashes(); len(flashes) > 0 {
		data["messages"] = flashes
	}

	errSession := getErrSession(r)

	if flashes := errSession.Flashes(); len(flashes) > 0 {
		data["errors"] = flashes
	}

	_ = session.Save(r, w)
	_ = errSession.Save(r, w)

	opts, err := raceManager.LoadServerOptions()

	if err != nil {
		return err
	}

	data["server_status"] = AssettoProcess.IsRunning()
	data["server_event_type"] = AssettoProcess.EventType()
	data["server_name"] = opts.Name
	data["User"] = AccountFromRequest(r)
	data["IsHosted"] = IsHosted
	data["IsPremium"] = IsPremium
	data["MaxClientsOverride"] = MaxClientsOverride
	data["_request"] = r
	data["Debug"] = Debug

	t.Funcs(map[string]interface{}{
		"ReadAccess":   ReadAccess(r),
		"WriteAccess":  WriteAccess(r),
		"DeleteAccess": DeleteAccess(r),
		"AdminAccess":  AdminAccess(r),
		"LoggedIn":     LoggedIn(r),
	})

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

func ordinal(x int64) string {
	suffix := "th"
	switch x % 10 {
	case 1:
		if x%100 != 11 {
			suffix = "st"
		}
	case 2:
		if x%100 != 12 {
			suffix = "nd"
		}
	case 3:
		if x%100 != 13 {
			suffix = "rd"
		}
	}

	return suffix
}

type AssetHelper struct {
	baseURL *url.URL
	prefix  string
	suffix  string
	query   map[string]string
}

func NewAssetHelper(baseURL, prefix, suffix string, query map[string]string) *AssetHelper {
	u, err := url.Parse(baseURL)

	if err != nil {
		panic("invalid base url: " + baseURL)
	}

	return &AssetHelper{
		baseURL: u,
		prefix:  prefix,
		suffix:  suffix,
		query:   query,
	}
}

func (a *AssetHelper) GetURL(location string) string {
	u, err := url.Parse(location)

	if err != nil {
		return location
	}

	u.Scheme = a.baseURL.Scheme
	u.Host = a.baseURL.Host
	u.Path = path.Join(a.prefix, u.Path, a.suffix)

	q := u.Query()

	for k, v := range a.query {
		q.Set(k, v)
	}

	// rebuild query
	u.RawQuery = q.Encode()

	return u.String()
}
