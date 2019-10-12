package servermanager

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cj123/ini"
	"github.com/sirupsen/logrus"
)

type FilterError string

func (f FilterError) Error() string {
	return string(f)
}

type RaceWeekendSessionToSessionFilter struct {
	IsPreview bool

	ResultStart int
	ResultEnd   int

	NumEntrantsToReverse int

	EntryListStart int

	SortType string

	ForceUseTyreFromFastestLap bool
}

func reverseEntrants(numToReverse int, entrants []*RaceWeekendSessionEntrant) {
	if numToReverse == 0 {
		return
	}

	if numToReverse > len(entrants) {
		numToReverse = len(entrants)
	}

	var toReverse []*RaceWeekendSessionEntrant

	if numToReverse > 0 {
		toReverse = entrants[:numToReverse]
	} else {
		toReverse = entrants
	}

	for i := len(toReverse)/2 - 1; i >= 0; i-- {
		opp := len(toReverse) - 1 - i
		toReverse[i], toReverse[opp] = toReverse[opp], toReverse[i]
	}

	for i := 0; i < len(toReverse); i++ {
		entrants[i] = toReverse[i]
	}
}

// Filter takes a set of RaceWeekendSessionEntrants formed by the results of the parent session and filters them into a child session entry list.
func (f RaceWeekendSessionToSessionFilter) Filter(raceWeekend *RaceWeekend, parentSession, childSession *RaceWeekendSession, parentSessionResults []*RaceWeekendSessionEntrant, childSessionEntryList *RaceWeekendEntryList) error {
	if parentSession.Completed() {
		sorter := GetRaceWeekendEntryListSort(f.SortType)

		parentSession.NumEntrantsToReverse = f.NumEntrantsToReverse

		// race weekend session is completed and has a valid sorter, use it to sort results before filtering.
		if err := sorter(parentSession, parentSessionResults); err != nil {
			return err
		}
	}

	resultStart, resultEnd, entryListStart := f.ResultStart, f.ResultEnd, f.EntryListStart

	resultStart--
	entryListStart--

	if resultStart > len(parentSessionResults) {
		return nil
	}

	if resultEnd > len(parentSessionResults) {
		resultEnd = len(parentSessionResults)
	}

	split := parentSessionResults[resultStart:resultEnd]

	if !parentSession.Completed() {
		reverseEntrants(f.NumEntrantsToReverse, split)
	}

	splitIndex := 0

	for pitBox := entryListStart; pitBox < entryListStart+(resultEnd-resultStart); pitBox++ {
		entrant := split[splitIndex]
		entrant.SessionID = parentSession.ID

		if !f.IsPreview && parentSession.Completed() && f.ForceUseTyreFromFastestLap {
			// find the tyre from the entrants fastest lap
			fastestLap := entrant.SessionResults.GetDriversFastestLap(entrant.Car.GetGUID(), entrant.Car.GetCar())

			if fastestLap == nil {
				logrus.Warnf("could not find fastest lap for entrant %s (%s). will not lock their tyre choice.", entrant.Car.GetName(), entrant.Car.GetGUID())
			} else {
				err := raceWeekend.buildLockedTyreSetup(entrant, fastestLap)

				if err != nil {
					logrus.WithError(err).Errorf("could not build locked tyre setup for entrant %s (%s)", entrant.Car.GetName(), entrant.Car.GetGUID())
				}
			}
		}

		childSessionEntryList.AddInPitBox(entrant, pitBox)

		splitIndex++
	}

	return nil
}

func (rw *RaceWeekend) buildLockedTyreSetup(entrant *RaceWeekendSessionEntrant, fastestLap *SessionLap) error {
	tyreIndex, err := findTyreIndex(entrant.Car.Model, fastestLap.Tyre)

	if err != nil {
		return err
	}

	entryList := rw.GetEntryList()

	var setup *ini.File

	for _, raceWeekendEntrant := range entryList {
		if raceWeekendEntrant.GUID == entrant.Car.GetGUID() && raceWeekendEntrant.FixedSetup != "" {
			setup, err = ini.Load(filepath.Join(ServerInstallPath, "setups", raceWeekendEntrant.FixedSetup))

			if err != nil {
				return err
			}

			break
		}
	}

	if setup == nil {
		// no fixed setup was specified
		// write out a temp ini setup file for this car + player.
		setup = ini.NewFile([]ini.DataSource{nil}, ini.LoadOptions{
			IgnoreInlineComment: true,
		})

		_, err = setup.NewSection("DEFAULT")

		if err != nil {
			return err
		}

		car, err := setup.NewSection("CAR")

		if err != nil {
			return err
		}

		_, err = car.NewKey("MODEL", entrant.Car.Model)

		if err != nil {
			return err
		}
	}

	tyres, err := setup.NewSection("TYRES")

	if err != nil {
		return err
	}

	_, err = tyres.NewKey("VALUE", fmt.Sprintf("%d", tyreIndex))

	if err != nil {
		return err
	}

	setupFilePath := filepath.Join(entrant.Car.Model, "locked_tyres", fmt.Sprintf("race_weekend_session_%s_%s.ini", entrant.Car.GetGUID(), entrant.SessionID.String()))

	fullSaveFilepath := filepath.Join(ServerInstallPath, "setups", setupFilePath)

	if err := os.MkdirAll(filepath.Dir(fullSaveFilepath), 0755); err != nil {
		return err
	}

	if err := setup.SaveTo(fullSaveFilepath); err != nil {
		return err
	}

	entrant.OverrideSetupFile = setupFilePath

	return nil
}
