package servermanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/JustaPenguin/assetto-server-manager/cmd/server-manager/static"

	"github.com/cj123/ini"
	"github.com/dimchansky/utfbom"
	"github.com/go-chi/chi"
	"github.com/jpillora/longestcommon"
	"github.com/nfnt/resize"
	"github.com/sirupsen/logrus"
)

type Track struct {
	Name    string
	Layouts []string

	CalculatedPrettyName string

	MetaData TrackMetaData
}

const defaultTrackURL = "/static/img/no-preview-general.png"

func (t Track) GetImagePath() string {
	if len(t.Layouts) == 0 {
		return defaultTrackURL
	}

	for _, layout := range t.Layouts {
		if layout == defaultLayoutName || layout == "" {
			return filepath.ToSlash(filepath.Join("content", "tracks", t.Name, "ui", "preview.png"))
		}
	}

	return filepath.ToSlash(filepath.Join("content", "tracks", t.Name, "ui", t.Layouts[0], "preview.png"))
}

func LoadTrackMetaDataFromName(name string) (*TrackMetaData, error) {
	metaDataFile := filepath.Join(ServerInstallPath, "content", "tracks", name, "ui")

	metaDataFile = filepath.Join(metaDataFile, trackMetaDataName)

	f, err := os.Open(metaDataFile)

	if err != nil && os.IsNotExist(err) {
		return &TrackMetaData{}, nil
	} else if err != nil {
		return nil, err
	}

	defer f.Close()

	var trackMetaData *TrackMetaData

	err = json.NewDecoder(utfbom.SkipOnly(f)).Decode(&trackMetaData)

	if err != nil {
		return nil, err
	}

	return trackMetaData, nil
}

func (t *Track) LoadMetaData() error {
	trackMetaData, err := LoadTrackMetaDataFromName(t.Name)

	if err != nil {
		return err
	}

	t.MetaData = *trackMetaData

	return nil
}

func (t Track) PrettyName() string {
	if t.CalculatedPrettyName != "" {
		return t.CalculatedPrettyName
	}

	return prettifyName(t.Name, false)
}

func (t Track) IsPaidDLC() bool {
	if _, ok := isTrackPaidDLC[t.Name]; ok {
		return isTrackPaidDLC[t.Name]
	}

	return false
}

func (t Track) IsMod() bool {
	_, ok := isTrackPaidDLC[t.Name]

	return !ok
}

const defaultLayoutName = "<default>"

func (t *Track) LayoutsCSV() string {
	if t.Layouts == nil {
		return "Default"
	}

	return strings.Join(t.Layouts, ", ")
}

func trackLayoutURL(track, layout string) string {
	var layoutPath string

	if layout == "" || layout == defaultLayoutName {
		layoutPath = filepath.Join("content", "tracks", track, "ui", "preview.png")
	} else {
		layoutPath = filepath.Join("content", "tracks", track, "ui", layout, "preview.png")
	}

	// look to see if the track preview image exists
	_, err := os.Stat(filepath.Join(ServerInstallPath, layoutPath))

	if err != nil {
		return defaultTrackURL
	}

	return "/" + filepath.ToSlash(layoutPath)
}

const trackInfoJSONName = "ui_track.json"
const trackMetaDataName = "meta_data.json"

type TrackInfo struct {
	Name        string      `json:"name"`
	City        string      `json:"city"`
	Country     string      `json:"country"`
	Description string      `json:"description"`
	Geotags     []string    `json:"geotags"`
	Length      string      `json:"length"`
	Pitboxes    json.Number `json:"pitboxes"`
	Run         string      `json:"run"`
	Tags        []string    `json:"tags"`
	Width       string      `json:"width"`
}

type TrackMetaData struct {
	DownloadURL string `json:"downloadURL"`
	Notes       string `json:"notes"`
}

func (tmd *TrackMetaData) Save(name string) error {
	uiDirectory := filepath.Join(ServerInstallPath, "content", "tracks", name, "ui")

	err := os.MkdirAll(uiDirectory, 0755)

	if err != nil {
		return err
	}

	f, err := os.Create(filepath.Join(uiDirectory, trackMetaDataName))

	if err != nil {
		return err
	}

	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "   ")

	return enc.Encode(tmd)
}

func GetTrackInfo(name, layout string) (*TrackInfo, error) {
	uiDataFile := filepath.Join(ServerInstallPath, "content", "tracks", name, "ui")

	if layout != "" && layout != defaultLayoutName {
		uiDataFile = filepath.Join(uiDataFile, layout)
	}

	uiDataFile = filepath.Join(uiDataFile, trackInfoJSONName)

	data, err := ioutil.ReadFile(uiDataFile)

	if err != nil {
		return nil, err
	}

	data = bytes.ReplaceAll(data, []byte("\r"), []byte(""))
	data = bytes.ReplaceAll(data, []byte("\n"), []byte(""))

	var trackInfo *TrackInfo

	err = json.NewDecoder(utfbom.SkipOnly(bytes.NewBuffer(data))).Decode(&trackInfo)

	return trackInfo, err
}

type TracksHandler struct {
	*BaseHandler

	trackManager *TrackManager
}

func NewTracksHandler(baseHandler *BaseHandler, trackManager *TrackManager) *TracksHandler {
	return &TracksHandler{
		BaseHandler:  baseHandler,
		trackManager: trackManager,
	}
}

type trackListTemplateVars struct {
	BaseTemplateVars

	Tracks []Track
}

func (th *TracksHandler) list(w http.ResponseWriter, r *http.Request) {
	tracks, err := th.trackManager.ListTracks()

	if err != nil {
		logrus.WithError(err).Errorf("could not get track list")
	}

	th.viewRenderer.MustLoadTemplate(w, r, "content/tracks.html", &trackListTemplateVars{
		Tracks: tracks,
	})
}

func (th *TracksHandler) delete(w http.ResponseWriter, r *http.Request) {
	trackName := chi.URLParam(r, "name")
	tracksPath := filepath.Join(ServerInstallPath, "content", "tracks")

	existingTracks, err := th.trackManager.ListTracks()

	if err != nil {
		logrus.WithError(err).Errorf("could not get track list")
		AddErrorFlash(w, r, "couldn't get track list")
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	var found bool

	for _, track := range existingTracks {
		if track.Name == trackName {
			// Delete track
			found = true

			err := os.RemoveAll(filepath.Join(tracksPath, trackName))

			if err != nil {
				found = false
				logrus.WithError(err).Errorf("could not remove track files")
			}

			for _, layout := range track.Layouts {
				clearFromTrackInfoCache(track.Name, layout)
			}

			break
		}
	}

	if found {
		// confirm deletion
		AddFlash(w, r, "Track successfully deleted!")
	} else {
		// inform track wasn't found
		AddErrorFlash(w, r, "Sorry, track could not be deleted.")
	}

	http.Redirect(w, r, "/tracks", http.StatusFound)
}

func (th *TracksHandler) view(w http.ResponseWriter, r *http.Request) {
	trackName := chi.URLParam(r, "track_id")
	templateParams, err := th.trackManager.loadTrackDetailsForTemplate(trackName)

	if os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		logrus.WithError(err).Errorf("Could not load track details for: %s", trackName)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	th.viewRenderer.MustLoadTemplate(w, r, "content/track-details.html", templateParams)
}

func (th *TracksHandler) saveMetadata(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")

	if err := th.trackManager.UpdateTrackMetadata(name, r); err != nil {
		logrus.WithError(err).Errorf("Could not update track metadata for %s", name)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, "Track metadata updated successfully!")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (th *TracksHandler) trackImage(w http.ResponseWriter, r *http.Request) {
	track := chi.URLParam(r, "track")
	layout := chi.URLParam(r, "layout")

	w.Header().Add("Content-Type", "image/png")
	n, err := th.trackManager.GetTrackImage(w, track, layout)

	if err != nil {
		image := static.FSMustByte(false, "/img/no-preview-general.png")
		_, _ = w.Write(image)
	} else {
		w.Header().Add("Content-Length", strconv.Itoa(int(n)))
	}
}

func (th *TracksHandler) trackInfo(w http.ResponseWriter, r *http.Request) {
	track := chi.URLParam(r, "track")
	layout := chi.URLParam(r, "layout")

	w.Header().Add("Content-Type", "application/json")

	_ = json.NewEncoder(w).Encode(trackInfo(track, layout))
}

type TrackManager struct {
}

func NewTrackManager() *TrackManager {
	return &TrackManager{}
}

type trackDetailsTemplateVars struct {
	BaseTemplateVars

	Track     *Track
	TrackInfo map[string]*TrackInfo
	Results   map[string][]SessionResults
}

func (tm *TrackManager) loadTrackDetailsForTemplate(trackName string) (*trackDetailsTemplateVars, error) {
	trackInfoMap := make(map[string]*TrackInfo)
	resultsMap := make(map[string][]SessionResults)

	track, err := tm.GetTrackFromName(trackName)

	if err != nil {
		return nil, err
	}

	err = track.LoadMetaData()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load meta data for track: %s", trackName)
	}

	for _, layout := range track.Layouts {
		trackInfo, err := GetTrackInfo(track.Name, layout)

		if err != nil {
			logrus.WithError(err).Errorf("Couldn't load track info for layout: %s, track: %s", layout, track.Name)
			continue
		}

		trackInfoMap[layout] = trackInfo

		results, err := tm.ResultsForLayout(track.Name, layout)

		if err != nil {
			logrus.WithError(err).Errorf("Couldn't load results for layout: %s, track: %s", layout, track.Name)
			continue
		}

		resultsMap[layout] = results
	}

	return &trackDetailsTemplateVars{
		BaseTemplateVars: BaseTemplateVars{},
		Track:            track,
		TrackInfo:        trackInfoMap,
		Results:          resultsMap,
	}, nil
}

func (tm *TrackManager) ResultsForLayout(trackName, layout string) ([]SessionResults, error) {
	results, err := ListAllResults()

	if err != nil {
		return nil, err
	}

	var out []SessionResults

	for _, result := range results {
		if result.TrackName == trackName && result.TrackConfig == layout {
			out = append(out, result)
		}
	}

	return out, nil
}

func (tm *TrackManager) ListTracks() ([]Track, error) {
	tracksPath := filepath.Join(ServerInstallPath, "content", "tracks")

	trackFiles, err := ioutil.ReadDir(tracksPath)

	if err != nil {
		return nil, err
	}

	var tracks []Track

	for _, trackFile := range trackFiles {
		track, err := tm.GetTrackFromName(trackFile.Name())

		if err != nil {
			continue
		}

		tracks = append(tracks, *track)
	}

	sort.Slice(tracks, func(i, j int) bool {
		return tracks[i].PrettyName() < tracks[j].PrettyName()
	})

	return tracks, nil
}

func (tm *TrackManager) decodeTrackImage(path string) (image.Image, error) {
	f, err := os.Open(path)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	data, _, err := image.Decode(f)

	if err != nil {
		return nil, err
	}

	return data, err
}

const (
	trackMapOverlayScale = 2.0
)

func (tm *TrackManager) GetTrackImage(w io.Writer, track, layout string) (int64, error) {
	trackContentPath := filepath.Join(ServerInstallPath, "content", "tracks", track, "ui")
	trackMapPath := filepath.Join(ServerInstallPath, "content", "tracks", track, "map.png")

	if layout != "" {
		trackContentPath = filepath.Join(trackContentPath, layout)
		trackMapPath = filepath.Join(ServerInstallPath, "content", "tracks", track, layout, "map.png")
	}

	trackImagePath := filepath.Join(trackContentPath, "preview.png")
	trackOutlinePath := filepath.Join(trackContentPath, "outline.png")

	combinedImageMapPath := filepath.Join(trackContentPath, "server-manager_preview.png")
	combinedImageMap, err := os.Open(combinedImageMapPath)

	if err == nil {
		defer combinedImageMap.Close()

		return io.Copy(w, combinedImageMap)
	} else if !os.IsNotExist(err) {
		return -1, err
	}

	trackImage, err := tm.decodeTrackImage(trackImagePath)

	if err != nil {
		return -1, err
	}

	trackMap, err := tm.decodeTrackImage(trackMapPath)

	if os.IsNotExist(err) {
		trackMap, err = tm.decodeTrackImage(trackOutlinePath)

		if err != nil {
			return -1, err
		}
	} else if err != nil {
		return -1, err
	}

	trackImageBounds := trackImage.Bounds()
	trackMapBounds := trackMap.Bounds()

	var resizedMap image.Image

	marginX, marginY := 10, 10

	if trackMapBounds.Dx() > trackMapBounds.Dy() {
		resizedMap = resize.Resize(uint(float64(trackImageBounds.Dx())/trackMapOverlayScale), 0, trackMap, resize.Bilinear)
	} else {
		resizedMap = resize.Resize(0, uint(float64(trackImageBounds.Dy())/trackMapOverlayScale), trackMap, resize.Bilinear)
		marginX = 20
		marginY = 20
	}

	combined := image.NewRGBA(trackImageBounds)
	draw.Draw(combined, trackImageBounds, trackImage, image.Point{}, draw.Src)
	draw.Draw(combined, trackImageBounds, resizedMap, image.Pt(-trackImageBounds.Dx()+resizedMap.Bounds().Dx()+marginX, -trackImageBounds.Dy()+resizedMap.Bounds().Dy()+marginY), draw.Over)

	combinedImageMap, err = os.Create(combinedImageMapPath)

	if err != nil {
		return -1, err
	}

	defer combinedImageMap.Close()

	if err := png.Encode(combinedImageMap, combined); err != nil {
		return -1, err
	}

	if _, err := combinedImageMap.Seek(0, 0); err != nil {
		return -1, err
	}

	return io.Copy(w, combinedImageMap)
}

func (tm *TrackManager) GetTrackFromName(name string) (*Track, error) {
	tracksPath := filepath.Join(ServerInstallPath, "content", "tracks")
	var layouts []string

	files, err := ioutil.ReadDir(filepath.Join(tracksPath, name))

	if err != nil {
		logrus.WithError(err).Errorf("Can't read folder: %s", name)
		return nil, err
	}

	// Check for multiple layouts, if tracks have data folders in the main directory then they only have one
	if len(files) > 1 {
		for _, layout := range files {
			if layout.IsDir() {
				switch layout.Name() {
				case "data":
					layouts = append(layouts, defaultLayoutName)
				case "ui":
					continue
				default:
					// valid layouts must contain a surfaces.ini
					_, err := os.Stat(filepath.Join(tracksPath, name, layout.Name(), "data", "surfaces.ini"))

					if os.IsNotExist(err) {
						continue
					} else if err != nil {
						return nil, err
					}

					layouts = append(layouts, layout.Name())
				}
			}
		}
	}

	track := &Track{Name: name, Layouts: layouts}

	var layoutNames []string

	for _, layout := range layouts {
		info := trackInfo(track.Name, layout)

		if info == nil {
			continue
		}

		layoutNames = append(layoutNames, info.Name)
	}

	if len(layoutNames) > 0 {
		if len(layoutNames) == 1 {
			track.CalculatedPrettyName = strings.Title(layoutNames[0])
		} else {
			sort.Slice(layoutNames, func(i, j int) bool {
				return len(layoutNames[i]) < len(layoutNames[j])
			})

			commonPrefix := strings.TrimRightFunc(longestcommon.Prefix(layoutNames), func(r rune) bool {
				return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != ']' && r != ')' && r != '}'
			})

			if commonPrefix != "" {
				track.CalculatedPrettyName = strings.Title(commonPrefix)
			}
		}
	}

	return track, nil
}

func (tm *TrackManager) UpdateTrackMetadata(name string, r *http.Request) error {
	track, err := tm.GetTrackFromName(name)

	if err != nil {
		return err
	}

	track.MetaData.Notes = r.FormValue("Notes")
	track.MetaData.DownloadURL = r.FormValue("DownloadURL")

	return track.MetaData.Save(name)
}

type TrackDataGateway interface {
	TrackInfo(name, layout string) (*TrackInfo, error)
	TrackMap(name, layout string) (*TrackMapData, error)
}

type filesystemTrackData struct{}

func (filesystemTrackData) TrackMap(name, layout string) (*TrackMapData, error) {
	return LoadTrackMapData(name, layout)
}

func (filesystemTrackData) TrackInfo(name, layout string) (*TrackInfo, error) {
	trackInfo, err := GetTrackInfo(name, layout)

	if err != nil {
		logrus.WithError(err).Errorf("Could not load track info")

		return &TrackInfo{
			Name:    trackSummary(name, layout),
			City:    "Unknown",
			Country: "Unknown",
		}, nil
	}

	return trackInfo, err
}

type TrackMapData struct {
	Width       float64 `ini:"WIDTH" json:"width"`
	Height      float64 `ini:"HEIGHT" json:"height"`
	Margin      float64 `ini:"MARGIN" json:"margin"`
	ScaleFactor float64 `ini:"SCALE_FACTOR" json:"scale_factor"`
	OffsetX     float64 `ini:"X_OFFSET" json:"offset_x"`
	OffsetZ     float64 `ini:"Z_OFFSET" json:"offset_y"`
	DrawingSize float64 `ini:"DRAWING_SIZE" json:"drawing_size"`
}

func LoadTrackMapData(track, trackLayout string) (*TrackMapData, error) {
	p := filepath.Join(ServerInstallPath, "content", "tracks", track)

	if trackLayout != "" {
		p = filepath.Join(p, trackLayout)
	}

	p = filepath.Join(p, "data", "map.ini")

	f, err := os.Open(p)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	i, err := ini.Load(f)

	if err != nil {
		return nil, err
	}

	s, err := i.GetSection("PARAMETERS")

	if err != nil {
		return nil, err
	}

	var mapData TrackMapData

	if err := s.MapTo(&mapData); err != nil {
		return nil, err
	}

	return &mapData, nil
}

func TrackMapImageURL(track, trackLayout string) string {
	p := "/content/tracks/" + track

	if trackLayout != "" {
		p += "/" + trackLayout
	}

	return p + "/map.png"
}

func LoadTrackMapImage(track, trackLayout string) (image.Image, error) {
	p := filepath.Join(ServerInstallPath, "content", "tracks", track)

	if trackLayout != "" {
		p = filepath.Join(p, trackLayout)
	}

	p = filepath.Join(p, "map.png")

	f, err := os.Open(p)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	return png.Decode(f)
}

// disableDRSFile is a file with a tiny DRS zone that is too small to activate DRS in.
const disableDRSFile = `
[ZONE_0]
DETECTION=0.899
START=0
END=0.0001 
`

const (
	drsZonesFilename       = "drs_zones.ini"
	drsZonesBackupFilename = "drs_zones.ini.orig"
)

func ToggleDRSForTrack(track, layout string, drsEnabled bool) error {
	trackPath := filepath.Join(ServerInstallPath, "content", "tracks", track, layout, "data")
	drsBackupFile := filepath.Join(trackPath, drsZonesBackupFilename)
	drsFile := filepath.Join(trackPath, drsZonesFilename)

	// if DRS is enabled
	if drsEnabled {
		// if the backup file exists, then rename it back into place
		if _, err := os.Stat(drsBackupFile); err == nil {
			logrus.Infof("Enabling DRS for %s (%s)", track, layout)
			err := os.Rename(drsBackupFile, drsFile)

			if err != nil && !os.IsNotExist(err) {
				return err
			}

			return nil
		} else if os.IsNotExist(err) {
			// there is no backup file. read the existing DRS file. if it's equal to disableDRSFile then we just want to delete it.
			currentDRSContents, err := ioutil.ReadFile(drsFile)

			if os.IsNotExist(err) {
				logrus.Infof("Track: %s (%s) has no drs file. DRS anywhere will be enabled.", track, layout)
				return nil
			} else if err != nil {
				return err
			}

			if string(currentDRSContents) == disableDRSFile {
				// the track has no original drs_zones.ini, just remove our file.
				logrus.Infof("Track: %s (%s) has no drs file. DRS anywhere will be enabled.", track, layout)
				err := os.Remove(drsFile)

				if err != nil && !os.IsNotExist(err) {
					return err
				}

				return nil
			}

			return nil
		} else { // err != nil
			return err
		}
	} else {
		logrus.Infof("Disabling DRS for: %s (%s)", track, layout)

		if _, err := os.Stat(drsBackupFile); os.IsNotExist(err) {
			// drs is not enabled, move the drs_zones file to backup
			if err := os.Rename(drsFile, drsBackupFile); err != nil && !os.IsNotExist(err) {
				return err
			}
		} else if err != nil {
			return err
		}

		// now write the disabled-drs file
		return ioutil.WriteFile(drsFile, []byte(disableDRSFile), 0644)
	}
}

func trackSummary(track, layout string) string {
	info := trackInfo(track, layout)

	if info != nil {
		return info.Name
	}

	track = prettifyName(track, false)

	if layout != "" {
		track += fmt.Sprintf(" (%s)", prettifyName(layout, true))
	}

	return track
}

func trackDownloadLink(track string) string {
	metaData, err := LoadTrackMetaDataFromName(track)

	if err != nil {
		return ""
	}

	return metaData.DownloadURL
}

var isTrackPaidDLC = map[string]bool{
	"ks_barcelona":        true,
	"ks_black_cat_county": false,
	"ks_brands_hatch":     true,
	"ks_drag":             false,
	"ks_highlands":        false,
	"ks_laguna_seca":      false,
	"ks_monza66":          false,
	"ks_nordschleife":     true,
	"ks_nurburgring":      false,
	"ks_red_bull_ring":    true,
	"ks_silverstone":      false,
	"ks_silverstone1967":  false,
	"ks_vallelunga":       false,
	"ks_zandvoort":        false,
	"monza":               false,
	"mugello":             false,
	"magione":             false,
	"drift":               false,
	"imola":               false,
	"spa":                 false,
	"trento-bondone":      false,
}
