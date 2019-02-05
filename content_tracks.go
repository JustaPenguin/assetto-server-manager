package servermanager

import (
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"
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
