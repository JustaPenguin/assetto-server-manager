package servermanager

import "io/ioutil"

type Track struct {
	Name    string
	Layouts []string
}

func ListTracks() ([]Track, error) {
	tracksPath := ServerInstallPath + "/content/tracks"

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

	return tracks, nil
}

func (t *Track) LayoutsCSV() string {
	var layoutsCSV string

	if t.Layouts == nil {
		layoutsCSV = "Default"
	} else {
		for i, layout := range t.Layouts {
			if i == len(t.Layouts)-1 {
				layoutsCSV = layoutsCSV + layout
			} else {
				layoutsCSV = layoutsCSV + layout + ", "
			}
		}
	}

	return layoutsCSV
}
