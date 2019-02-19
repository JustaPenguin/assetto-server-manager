package servermanager

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

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

func ListTracks() ([]Track, error) {
	tracksPath := filepath.Join(ServerInstallPath, "content", "tracks")

	trackFiles, err := ioutil.ReadDir(tracksPath)

	if err != nil {
		return nil, err
	}

	var tracks []Track

	for _, trackFile := range trackFiles {
		var layouts []string

		files, err := ioutil.ReadDir(tracksPath + "/" + trackFile.Name())

		if err != nil {
			return nil, err
		}

		// Check for multiple layouts, if tracks have data folders in the main directory then they only have one
		if len(files) > 1 {
			for _, layout := range files {
				if layout.IsDir() {
					if layout.Name() == "data" {
						// track only has one layout
						layouts = nil
						break
					} else if layout.Name() == "ui" {
						// ui folder, not a layout
						continue
					} else {
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
	Length      json.Number `json:"length"`
	Pitboxes    json.Number `json:"pitboxes"`
	Run         string      `json:"run"`
	Tags        []string    `json:"tags"`
	Width       json.Number `json:"width"`
}

func GetTrackInfo(name, layout string) (*TrackInfo, error) {
	uiDataFile := filepath.Join(ServerInstallPath, "content", "tracks", name, "ui")

	if layout != "" {
		uiDataFile = filepath.Join(uiDataFile, layout)
	}

	uiDataFile = filepath.Join(uiDataFile, trackInfoJSONName)

	f, err := os.Open(uiDataFile)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	var trackInfo *TrackInfo

	err = json.NewDecoder(f).Decode(&trackInfo)

	return trackInfo, err
}

func tracksHandler(w http.ResponseWriter, r *http.Request) {
	tracks, err := ListTracks()

	if err != nil {
		logrus.Errorf("could not get track list, err: %s", err)
	}

	ViewRenderer.MustLoadTemplate(w, r, filepath.Join("content", "tracks.html"), map[string]interface{}{
		"tracks": tracks,
	})
}

func apiTrackUploadHandler(w http.ResponseWriter, r *http.Request) {
	uploadHandler(w, r, "Track")
}

func trackDeleteHandler(w http.ResponseWriter, r *http.Request) {
	trackName := chi.URLParam(r, "name")
	tracksPath := filepath.Join(ServerInstallPath, "content", "tracks")

	existingTracks, err := ListTracks()

	if err != nil {
		logrus.Errorf("could not get track list, err: %s", err)

		AddErrFlashQuick(w, r, "couldn't get track list")

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
		AddFlashQuick(w, r, "Track successfully deleted!")
	} else {
		// inform track wasn't found
		AddErrFlashQuick(w, r, "Sorry, track could not be deleted.")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
