package servermanager

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cj123/ini"
	"github.com/dimchansky/utfbom"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

type Track struct {
	Name    string
	Layouts []string
}

func (t Track) PrettyName() string {
	return prettifyName(t.Name, false)
}

const defaultLayoutName = "<default>"

func ListTracks() ([]Track, error) {
	tracksPath := filepath.Join(ServerInstallPath, "content", "tracks")

	trackFiles, err := ioutil.ReadDir(tracksPath)

	if err != nil {
		return nil, err
	}

	var tracks []Track

	for _, trackFile := range trackFiles {
		var layouts []string

		files, err := ioutil.ReadDir(filepath.Join(tracksPath, trackFile.Name()))

		if err != nil {
			logrus.Errorf("Can't read folder: %s, err: %s", trackFile.Name(), err)
			continue
		}

		// Check for multiple layouts, if tracks have data folders in the main directory then they only have one
		if len(files) > 1 {
			for _, layout := range files {
				if layout.IsDir() {
					if layout.Name() == "data" {
						layouts = append(layouts, defaultLayoutName)
					} else if layout.Name() == "ui" {
						// ui folder, not a layout
						continue
					} else {
						// valid layouts must contain a surfaces.ini
						_, err := os.Stat(filepath.Join(tracksPath, trackFile.Name(), layout.Name(), "data", "surfaces.ini"))

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

		tracks = append(tracks, Track{
			Name:    trackFile.Name(),
			Layouts: layouts,
		})
	}

	sort.Slice(tracks, func(i, j int) bool {
		return tracks[i].PrettyName() < tracks[j].PrettyName()
	})

	return tracks, nil
}

func (t *Track) LayoutsCSV() string {
	if t.Layouts == nil {
		return "Default"
	}

	return strings.Join(t.Layouts, ", ")
}

const trackInfoJSONName = "ui_track.json"

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

func GetTrackInfo(name, layout string) (*TrackInfo, error) {
	uiDataFile := filepath.Join(ServerInstallPath, "content", "tracks", name, "ui")

	if layout != "" && layout != defaultLayoutName {
		uiDataFile = filepath.Join(uiDataFile, layout)
	}

	uiDataFile = filepath.Join(uiDataFile, trackInfoJSONName)

	f, err := os.Open(uiDataFile)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	var trackInfo *TrackInfo

	err = json.NewDecoder(utfbom.SkipOnly(f)).Decode(&trackInfo)

	return trackInfo, err
}

type TracksHandler struct {
	*BaseHandler
}

func NewTracksHandler(baseHandler *BaseHandler) *TracksHandler {
	return &TracksHandler{
		BaseHandler: baseHandler,
	}
}

type trackListTemplateVars struct {
	BaseTemplateVars

	Tracks []Track
}

func (th *TracksHandler) list(w http.ResponseWriter, r *http.Request) {
	tracks, err := ListTracks()

	if err != nil {
		logrus.Errorf("could not get track list, err: %s", err)
	}

	th.viewRenderer.MustLoadTemplate(w, r, "content/tracks.html", &trackListTemplateVars{
		Tracks: tracks,
	})
}

func (th *TracksHandler) delete(w http.ResponseWriter, r *http.Request) {
	trackName := chi.URLParam(r, "name")
	tracksPath := filepath.Join(ServerInstallPath, "content", "tracks")

	existingTracks, err := ListTracks()

	if err != nil {
		logrus.Errorf("could not get track list, err: %s", err)
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
				logrus.Errorf("could not remove track files, err: %s", err)
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

	http.Redirect(w, r, r.Referer(), http.StatusFound)
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
	return GetTrackInfo(name, layout)
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
	} else {
		track := prettifyName(track, false)

		if layout != "" {
			track += fmt.Sprintf(" (%s)", prettifyName(layout, true))
		}

		return track
	}
}
