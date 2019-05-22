package main

import servermanager "github.com/cj123/assetto-server-manager"

func main() {
	old := servermanager.NewBoltStore()
}

func convertStore(old servermanager.Store, new servermanager.Store) error {
	// custom races
	oldRaces, err := old.ListCustomRaces()

	if err != nil {
		return err
	}

	for _, race := range oldRaces {
		err := new.UpsertCustomRace(race)

		if err != nil {
			return err
		}
	}

	// entrants
	oldEntrants, err := old.ListEntrants()

	if err != nil {
		return err
	}

	for _, entrant := range oldEntrants {
		err := new.UpsertEntrant(*entrant)

		if err != nil {
			return err
		}
	}

	// server options
	oldOpts, err := old.LoadServerOptions()

	if err != nil {
		return err
	}

	err = new.UpsertServerOptions(oldOpts)

	if err != nil {
		return err
	}

	// championships
	oldChamps, err := old.ListChampionships()

	if err != nil {
		return err
	}

	for _, champ := range oldChamps {
		err := new.UpsertChampionship(champ)

		if err != nil {
			return err
		}
	}

	// live timings
	oldFrames, err := old.ListPrevFrames()

	if err != nil {
		return err
	}

	err = old.UpsertLiveFrames(oldFrames)

	if err != nil {
		return err
	}

	// accounts
	oldAccounts, err := old.ListAccounts()

	if err != nil {
		return err
	}

	for _, acc := range oldAccounts {
		err := new.UpsertAccount(acc)

		if err != nil {
			return err
		}
	}

	return nil
}