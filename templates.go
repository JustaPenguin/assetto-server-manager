package servermanager

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net"
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
	"github.com/getsentry/raven-go"
	"github.com/go-chi/chi"
	"github.com/mattn/go-zglob"
	"github.com/sirupsen/logrus"
)

// BuildVersion is the time Server Manager was built at
var BuildVersion string

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

var UseShortenedDriverNames = true

func shortenDriverName(name string) string {
	nameParts := strings.Split(name, " ")

	if len(nameParts) > 1 && len(nameParts[len(nameParts)-1]) > 1 {
		nameParts[len(nameParts)-1] = nameParts[len(nameParts)-1][:1] + "."
	}

	return strings.Join(nameParts, " ")
}

func driverName(name string) string {
	if UseShortenedDriverNames {
		return shortenDriverName(name)
	}

	return name
}

func driverInitials(name string) string {
	if UseShortenedDriverNames {
		nameParts := strings.Split(name, " ")

		if len(nameParts) == 1 {
			return name
		}

		for i := range nameParts {
			if len(nameParts[i]) > 0 {
				nameParts[i] = nameParts[i][:1]
			}
		}

		return strings.ToUpper(strings.Join(nameParts, ""))
	}

	nameParts := strings.Split(name, " ")

	if len(nameParts) > 0 && len(nameParts[len(nameParts)-1]) >= 3 {
		return strings.ToUpper(nameParts[len(nameParts)-1][:3])
	}

	return strings.ToUpper(name)
}

// Renderer is the template engine.
type Renderer struct {
	store   Store
	process ServerProcess
	loader  TemplateLoader

	templates map[string]*template.Template

	reload bool
	mutex  sync.Mutex
}

func NewRenderer(loader TemplateLoader, store Store, process ServerProcess, reload bool) (*Renderer, error) {
	tr := &Renderer{
		store:   store,
		process: process,
		loader:  loader,

		templates: make(map[string]*template.Template),
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
	funcs["htmlAttr"] = func(s string) template.HTMLAttr {
		return template.HTMLAttr(s)
	}
	funcs["ordinal"] = ordinal
	funcs["prettify"] = prettifyName
	funcs["carList"] = carList
	funcs["jsonEncode"] = jsonEncode
	funcs["varSplit"] = varSplit
	funcs["timeFormat"] = timeFormat
	funcs["dateFormat"] = dateFormat
	funcs["timeZone"] = timeZone
	funcs["hourAndZone"] = hourAndZoneFormat
	funcs["localFormatHourAndZone"] = localFormatHelperHourAndZone
	funcs["addTime"] = addTime
	funcs["isBefore"] = isBefore
	funcs["trackInfo"] = trackInfo
	funcs["multiplyFloats"] = multiplyFloats
	funcs["stripGeotagCrap"] = stripGeotagCrap
	funcs["ReadAccess"] = dummyAccessFunc
	funcs["WriteAccess"] = dummyAccessFunc
	funcs["DeleteAccess"] = dummyAccessFunc
	funcs["AdminAccess"] = dummyAccessFunc
	funcs["LoggedIn"] = dummyAccessFunc
	funcs["classColor"] = ChampionshipClassColor
	funcs["carSkinURL"] = carSkinURL
	funcs["trackLayoutURL"] = trackLayoutURL
	funcs["stringArrayToCSV"] = stringArrayToCSV
	funcs["dict"] = templateDict
	funcs["asset"] = NewAssetHelper("/", "", "", map[string]string{"cb": BuildVersion}).GetURL
	funcs["SessionType"] = func(s string) SessionType { return SessionType(s) }
	funcs["Config"] = func() *Configuration { return config }
	funcs["Version"] = func() string { return BuildVersion }
	funcs["fullTimeFormat"] = fullTimeFormat
	funcs["localFormat"] = localFormatHelper
	funcs["driverName"] = driverName
	funcs["trustHTML"] = func(s string) template.HTML {
		return template.HTML(s)
	}
	funcs["formatDuration"] = formatDuration
	funcs["appendQuery"] = appendQuery
	funcs["ChangelogHTML"] = changelogHTML

	tr.templates, err = tr.loader.Templates(funcs)

	if err != nil {
		return err
	}

	return nil
}

var Changelog template.HTML

func changelogHTML() template.HTML {
	return Changelog
}

func appendQuery(r *http.Request, query, value string) string {
	q := r.URL.Query()
	q.Set(query, value)
	r.URL.RawQuery = q.Encode()

	return r.URL.String()
}

func formatDuration(d time.Duration, trimLeadingZeroes bool) string {
	hours := d.Hours()
	minutes := d.Minutes() - float64(int(hours)*60)
	seconds := d.Seconds() - float64(int(hours)*60*60) - float64(int(minutes)*60)

	duration := fmt.Sprintf("%02d:%02d:%06.3f", int(hours), int(minutes), seconds)

	if trimLeadingZeroes && strings.HasPrefix(duration, "00:") {
		return duration[3:]
	}

	return duration
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

func localFormatHelper(t time.Time) template.HTML {
	return template.HTML(fmt.Sprintf(`<span class="time-local" data-toggle="tooltip" data-time="%s" title="Translated to your timezone from %s">%s</span>`, t.Format(time.RFC3339), fullTimeFormat(t), fullTimeFormat(t)))
}

func localFormatHelperHourAndZone(t time.Time) template.HTML {
	return template.HTML(fmt.Sprintf(`<span class="time-local-kitchen" data-toggle="tooltip" data-time="%s" title="Translated to your timezone from %s">%s</span>`, t.Format(time.RFC3339), fullTimeFormat(t), fullTimeFormat(t)))
}

func timeFormat(t time.Time) string {
	return t.Format(time.Kitchen)
}

func dateFormat(t time.Time) string {
	return t.Format("02/01/2006")
}

func hourAndZoneFormat(t time.Time, plusMinutes int64) string {
	t = t.Add(time.Minute * time.Duration(plusMinutes))

	return t.Format("3:04 PM (MST)")
}

func addTime(t time.Time, plusMinutes int64) time.Time {
	return t.Add(time.Minute * time.Duration(plusMinutes))
}

func timeZone(t time.Time) string {
	name, _ := t.Zone()

	return name
}

func fullTimeFormat(t time.Time) string {
	return t.Format("Monday, January 2, 2006 3:04 PM (MST)")
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
	case []*SessionCar:
		carMap := make(map[string]bool)

		for _, entrant := range cars {
			carMap[entrant.Model] = true
		}

		for car := range carMap {
			split = append(split, car)
		}
	case EntryList:
		carMap := make(map[string]bool)

		for _, entrant := range cars {
			carMap[entrant.Model] = true
		}

		for car := range carMap {
			split = append(split, car)
		}
	default:
		panic("unknown type of cars: " + reflect.TypeOf(cars).String())
	}

	var out []string

	for _, s := range split {
		if s == AnyCarModel {
			continue
		}

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
		logrus.WithError(err).Errorf("Could not get track info for %s (%s)", track, layout)
		return nil
	}

	trackInfoCache[track+layout] = t

	return t
}

func multiplyFloats(a, b float64) float64 {
	return a * b
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
		/*for _, thing := range geoTags {
			println(thing)
		}*/
	} else if len(geoTags) == 1 {
		// dunno, some crazy format, just return
		return geoTags[0]
	}

	// Geotags of "lost" - a hamlet in Scotland
	if north {
		return "57.2050"
	}

	return "-3.0774"
}

var nameRegex = regexp.MustCompile(`^[A-Za-z]{0,5}[0-9]+`)

func prettifyName(s string, acronyms bool) string {
	if s == AnyCarModel {
		return "Any Car Model"
	}

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

func stringArrayToCSV(array []string) string {
	return strings.Join(array, ", ")
}

func jsonEncode(v interface{}) template.JS {
	buf := new(bytes.Buffer)

	_ = json.NewEncoder(buf).Encode(v)

	return template.JS(buf.String())
}

type TemplateVars interface {
	Get() *BaseTemplateVars
}

type BaseTemplateVars struct {
	Messages           []interface{}
	Errors             []interface{}
	ServerStatus       bool
	ServerEvent        RaceEvent
	ServerName         string
	CustomCSS          template.CSS
	User               *Account
	IsHosted           bool
	IsPremium          bool
	MaxClientsOverride int
	IsDarkTheme        bool
	Request            *http.Request
	Debug              bool
	MonitoringEnabled  bool
	SentryDSN          template.JSStr
	RecaptchaSiteKey   string
	WideContainer      bool
	OGImage            string
	ACSREnabled        bool
	BaseURLIsSet       bool
	BaseURLIsValid     bool
	ServerID           ServerID
}

func (b *BaseTemplateVars) Get() *BaseTemplateVars {
	return b
}

func (tr *Renderer) addData(w http.ResponseWriter, r *http.Request, vars TemplateVars) error {
	session := getSession(r)

	data := vars.Get()

	if flashes := session.Flashes(); len(flashes) > 0 {
		data.Messages = flashes
	}

	errSession := getErrSession(r)

	if flashes := errSession.Flashes(); len(flashes) > 0 {
		data.Errors = flashes
	}

	_ = session.Save(r, w)
	_ = errSession.Save(r, w)

	opts, err := tr.store.LoadServerOptions()

	if err != nil {
		return err
	}

	data.ServerStatus = tr.process.IsRunning()
	data.ServerEvent = tr.process.Event()
	data.ServerName = opts.Name
	data.CustomCSS = template.CSS(opts.CustomCSS)
	data.User = AccountFromRequest(r)
	data.IsHosted = IsHosted
	data.IsPremium = Premium()
	data.MaxClientsOverride = MaxClientsOverride
	data.IsDarkTheme = opts.DarkTheme == 1
	data.Request = r
	data.Debug = Debug
	data.MonitoringEnabled = config.Monitoring.Enabled
	data.SentryDSN = sentryJSDSN
	data.RecaptchaSiteKey = config.Championships.RecaptchaConfig.SiteKey
	data.BaseURLIsSet = baseURLIsSet()
	data.BaseURLIsValid = baseURLIsValid()
	data.ACSREnabled = opts.EnableACSR
	data.ServerID = serverID

	if Premium() {
		data.OGImage = opts.OGImage

		id := chi.URLParam(r, "championshipID")

		if id != "" {
			championship, err := tr.store.LoadChampionship(id)

			if err == nil && championship.OGImage != "" {
				data.OGImage = championship.OGImage
			}
		}
	}

	return nil
}

// LoadTemplate reads a template from templates and renders it with data to the given io.Writer
func (tr *Renderer) LoadTemplate(w http.ResponseWriter, r *http.Request, view string, vars TemplateVars) error {
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

	if vars == nil {
		vars = &BaseTemplateVars{}
	}

	if err := tr.addData(w, r, vars); err != nil {
		return err
	}

	t.Funcs(map[string]interface{}{
		"ReadAccess":   ReadAccess(r),
		"WriteAccess":  WriteAccess(r),
		"DeleteAccess": DeleteAccess(r),
		"AdminAccess":  AdminAccess(r),
		"LoggedIn":     LoggedIn(r),
	})

	return t.ExecuteTemplate(w, "base", vars)
}

// MustLoadTemplate asserts that a LoadTemplate call must succeed or be dealt with via the http.ResponseWriter
func (tr *Renderer) MustLoadTemplate(w http.ResponseWriter, r *http.Request, view string, vars TemplateVars) {
	err := tr.LoadTemplate(w, r, view, vars)

	if err != nil {
		if _, ok := err.(*net.OpError); !ok {
			// don't capture OpErrors, they flood sentry with non-errors
			raven.CaptureError(err, nil)
		}
		logrus.WithError(err).Errorf("Unable to load template: %s", view)
		http.Error(w, "unable to load template", http.StatusInternalServerError)
		return
	}
}

func (tr *Renderer) LoadPartial(w http.ResponseWriter, r *http.Request, partial string, vars TemplateVars) error {
	if tr.reload {
		// reload templates on every request if enabled, so
		// that we don't have to constantly restart the website
		err := tr.init()

		if err != nil {
			return err
		}
	}

	t, ok := tr.templates[filepath.ToSlash(partial)]

	if !ok {
		return errors.New("partial not found")
	}

	if vars == nil {
		vars = &BaseTemplateVars{}
	}

	if err := tr.addData(w, r, vars); err != nil {
		return err
	}

	t.Funcs(map[string]interface{}{
		"ReadAccess":   ReadAccess(r),
		"WriteAccess":  WriteAccess(r),
		"DeleteAccess": DeleteAccess(r),
		"AdminAccess":  AdminAccess(r),
		"LoggedIn":     LoggedIn(r),
	})

	return t.ExecuteTemplate(w, "partial", vars)
}

func (tr *Renderer) MustLoadPartial(w http.ResponseWriter, r *http.Request, partial string, vars TemplateVars) {
	err := tr.LoadPartial(w, r, partial, vars)

	if err != nil {
		logrus.WithError(err).Errorf("Unable to load partial: %s", partial)
		http.Error(w, "unable to load partial", http.StatusInternalServerError)
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
