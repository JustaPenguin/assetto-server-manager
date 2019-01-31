package servermanager

import (
	"fmt"
	"net/http"
)

var raceManager = &RaceManager{}

type RaceManager struct {

}

// Quick Race builds a quick race form
func (rm *RaceManager) QuickRace(r *http.Request) (map[string]interface{}, error) {
	cars, err := ListCars()

	if err != nil {
		return nil, err
	}

	tracks, err := ListTracks()

	if err != nil {
		return nil, err
	}

	var carNames, trackNames, trackLayouts []string

	for _, car := range cars {
		carNames = append(carNames, car.Name)
	}

	// @TODO eventually this will be loaded from somewhere
	currentRaceConfig := &ConfigIniDefault

	for _, track := range tracks {
		trackNames = append(trackNames, track.Name)

		for _, layout := range track.Layouts {
			trackLayouts = append(trackLayouts, fmt.Sprintf("%s:%s", track.Name, layout))
		}
	}


	for i, layout := range trackLayouts {
		if layout == fmt.Sprintf("%s:%s", currentRaceConfig.Server.CurrentRaceConfig.Track, currentRaceConfig.Server.CurrentRaceConfig.TrackLayout) {
			// mark the current track layout so the javascript can correctly set it up.
			trackLayouts[i] += ":current"
			break
		}
	}

	return map[string]interface{}{
		"CarOpts":         carNames,
		"TrackOpts":       trackNames,
		"TrackLayoutOpts": trackLayouts,
	}, nil
}