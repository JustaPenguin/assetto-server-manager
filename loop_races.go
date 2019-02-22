package servermanager

import (
	"github.com/cj123/assetto-server-manager/pkg/udp"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"time"
)

var sessionTypes []SessionType

func LoopRaces() {
	var i int
	ticker := time.NewTicker(30 * time.Second)

	for {
		select {
		case <-ticker.C:
			currentRace, _ := raceManager.CurrentRace()

			if currentRace != nil {
				break
			}

			_, _, looped, err := raceManager.ListCustomRaces()

			if err != nil {
				logrus.Errorf("couldn't list custom races, err: %s", err)
				return
			}

			if looped != nil {
				if i >= len(looped) {
					i = 0
				}

				// Reset the stored session types
				sessionTypes = []SessionType{}

				for sessionID := range looped[i].RaceConfig.Sessions {
					sessionTypes = append(sessionTypes, sessionID)
				}

				err := raceManager.StartCustomRace(looped[i].UUID.String(), true)

				if err != nil {
					logrus.Errorf("couldn't start auto loop custom race, err: %s", err)
					return
				}

				i++
			}
		}
	}

}

// callback check for udp end session, load result file, check session type against sessionTypes
// if session matches last session in sessionTypes then stop server and clear sessionTypes
func LoopCallbackFunc(message udp.Message) {
	switch a := message.(type) {
	case udp.EndSession:
		if sessionTypes == nil {
			logrus.Debugf("Session types == nil. ignoring end session callback")
			return
		}

		filename := filepath.Base(string(a))

		results, err := LoadResult(filename)

		if err != nil {
			logrus.Errorf("Could not read session results for %s, err: %s", filename, err)
			return
		}

		var endSession SessionType

		for _, session := range sessionTypes {
			if session == SessionTypeRace {
				endSession = SessionTypeRace
				break
			} else if session == SessionTypeQualifying {
				endSession = SessionTypeQualifying
			} else if session == SessionTypePractice && endSession != SessionTypeQualifying {
				endSession = SessionTypePractice
			} else if session == SessionTypeBooking && (endSession != SessionTypeQualifying && endSession != SessionTypePractice) {
				endSession = SessionTypeBooking
			}
		}

		logrus.Debugf("results type: %s, endSession: %s", results.Type, string(endSession))

		if results.Type == string(endSession) {
			logrus.Infof("Event end detected, stopping looped session.")

			sessionTypes = nil

			err := AssettoProcess.Stop()

			if err != nil {
				logrus.Errorf("Could not stop server, err: %s", err)
				return
			}
		}
	}
}
