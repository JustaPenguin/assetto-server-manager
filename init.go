package servermanager

import (
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func InitWithResolver(resolver *Resolver) error {
	store := resolver.ResolveStore()

	err := store.GetMeta(serverAccountOptionsMetaKey, &accountOptions)

	if err != nil && err != ErrValueNotSet {
		return err
	}

	opts, err := store.LoadServerOptions()

	if err != nil && err != ErrValueNotSet {
		return err
	}

	UseShortenedDriverNames = opts != nil && opts.UseShortenedDriverNames == 1

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	process := resolver.resolveServerProcess()
	championshipManager := resolver.resolveChampionshipManager()

	go func() {
		for range c {
			// ^C, handle it
			if process.IsRunning() {
				if process.EventType() == EventTypeChampionship {
					if err := championshipManager.StopActiveEvent(); err != nil {
						logrus.WithError(err).Errorf("Error stopping event")
					}
				} else {
					if err := process.Stop(); err != nil {
						logrus.WithError(err).Errorf("Could not stop server")
					}
				}

				if p, ok := process.(*AssettoServerProcess); ok {
					p.stopChildProcesses()
				}
			}

			os.Exit(0)
		}
	}()

	raceManager := resolver.resolveRaceManager()
	go raceManager.LoopRaces()

	err = raceManager.InitScheduledRaces()

	if err != nil {
		return err
	}

	err = championshipManager.InitScheduledChampionships()

	if err != nil {
		return err
	}

	carManager := resolver.resolveCarManager()

	err = carManager.CreateSearchIndex()

	if err != nil {
		return err
	}

	err = carManager.IndexAllCars()

	if err != nil {
		return err
	}

	return nil
}
