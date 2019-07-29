package servermanager

type RaceWeekendManager struct {
	raceManager *RaceManager
	store       Store
	process     ServerProcess
}

func NewRaceWeekendManager(raceManager *RaceManager, store Store, process ServerProcess) *RaceWeekendManager {
	return &RaceWeekendManager{
		raceManager: raceManager,
		store:       store,
		process:     process,
	}
}

func (rwm *RaceWeekendManager) StartEvent(raceWeekendID string, raceWeekendEventID string) error {
	raceWeekend, err := rwm.store.LoadRaceWeekend(raceWeekendID)

	if err != nil {
		return err
	}

	event, err := raceWeekend.FindEventByID(raceWeekendEventID)

	if err != nil {
		return err
	}

	entryList, err := event.GetEntryList(raceWeekend)

	if err != nil {
		return err
	}

	// @TODO replace normalEvent with something better here
	return rwm.raceManager.applyConfigAndStart(event.RaceConfig, entryList, false, normalEvent{})
}
