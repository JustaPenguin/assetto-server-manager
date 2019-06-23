package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/cj123/assetto-server-manager"
	"github.com/etcd-io/bbolt"
)

var (
	oldStore, newStore string
)

func init() {
	flag.StringVar(&oldStore, "old", "", "the store to convert (bolt format)")
	flag.StringVar(&newStore, "new", "", "the store to output (json format)")
	flag.Parse()
}

func main() {
	if oldStore == "" || newStore == "" {
		fmt.Println("you must specify a store. run with help args to find out more")
		os.Exit(1)
	}

	bdb, err := bbolt.Open(oldStore, 0755, nil)

	if err != nil {
		panic(err)
	}

	defer bdb.Close()

	old := servermanager.NewBoltStore(bdb).(*servermanager.BoltStore)
	old.ShowDeleted = true
	new := servermanager.NewJSONStore(newStore)

	err = convertStore(old, new)

	if err != nil {
		panic(err)
	}
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

	err = new.UpsertLiveFrames(oldFrames)

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

	// meta
	accOpts := &servermanager.ServerAccountOptions{
		IsOpen: false,
	}

	err = old.GetMeta("server-account-options", &accOpts)

	if err != nil {
		return err
	}

	err = new.SetMeta("server-account-options", accOpts)

	if err != nil {
		return err
	}

	var version int

	err = old.GetMeta("version", &version)

	if err != nil {
		return err
	}

	err = new.SetMeta("version", version)

	if err != nil {
		return err
	}

	return nil
}
