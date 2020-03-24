package servermanager

import "time"

func combineResults(results []*SessionResults) *SessionResults {
	output := &SessionResults{
		Cars:           nil,
		Events:         nil,
		Laps:           nil,
		Result:         nil,
		TrackConfig:    "",
		TrackName:      "",
		Type:           "",
		Date:           time.Time{},
		SessionFile:    "",
		ChampionshipID: "",
		RaceWeekendID:  "",
	}

	var trackName, trackConfig, championshipID, raceWeekendID string
	var sessionType SessionType

	for i, result := range results {

		if i == 0 {
			trackName = result.TrackName
			trackConfig = result.TrackConfig
			sessionType = result.Type
			championshipID = result.ChampionshipID
			raceWeekendID = result.RaceWeekendID
		} else {
			if result.TrackName != trackName || result.TrackConfig != trackConfig || result.Type != sessionType {
				// don't merge results from multiple tracks/layouts, or different session types
				continue
			}

			if result.ChampionshipID == championshipID {
				// if all from one championship keep the ID
				output.ChampionshipID = championshipID
			} else {
				// if not, clear it completely
				output.ChampionshipID = ""
			}

			if result.RaceWeekendID == raceWeekendID {
				// if all from one race weekend keep the ID
				output.RaceWeekendID = raceWeekendID
			} else {
				// if not, clear it completely
				output.RaceWeekendID = ""
			}
		}

		output.TrackName = trackName
		output.TrackConfig = trackConfig
		output.Type = sessionType
		output.Date = result.Date
		output.SessionFile = result.SessionFile

		cars:
		for _, car := range result.Cars {
			for _, existingCar := range output.Cars {
				if existingCar.GetGUID() == car.GetGUID() && existingCar.GetCar() == car.GetCar() {
					// car already added, skip
					continue cars
				}
			}

			output.Cars = append(output.Cars, car)
		}

		for _, event := range result.Events {
			output.Events = append(output.Events, event)
		}

		for _, lap := range result.Laps {
			output.Laps = append(output.Laps, lap)
		}

		// use fallbacksort to build result and sort
		output.FallBackSort()

		// @TODO write out session file?

	}

	return output
}