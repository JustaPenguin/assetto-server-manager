package servermanager

import (
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type ServerID string

var (
	serverID        ServerID
	serverIDMetaKey = "server_id"
)

func initServerID(store Store) error {
	err := store.GetMeta(serverIDMetaKey, &serverID)

	if err == ErrValueNotSet {
		serverID = ServerID(uuid.New().String())
		err = store.SetMeta(serverIDMetaKey, serverID)

		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	OpenAccount = &Account{
		Name:            "Free Access",
		Groups:          map[ServerID]Group{serverID: GroupRead},
		LastSeenVersion: BuildVersion,
		Theme:           ThemeDefault,
	}

	return nil
}

func InitWithResolver(resolver *Resolver) error {
	store := resolver.ResolveStore()

	err := store.GetMeta(serverAccountOptionsMetaKey, &accountOptions)

	if err != nil && err != ErrValueNotSet {
		return err
	}

	if err := initServerID(store); err != nil {
		return err
	}

	logrus.Infof("Server manager instance identifies as: %s", serverID)

	opts, err := store.LoadServerOptions()

	if err != nil && err != ErrValueNotSet {
		return err
	}

	UseShortenedDriverNames = opts != nil && opts.UseShortenedDriverNames == 1
	UseFallBackSorting = opts != nil && opts.FallBackResultsSorting == 1

	process := resolver.resolveServerProcess()
	championshipManager := resolver.resolveChampionshipManager()
	raceWeekendManager := resolver.resolveRaceWeekendManager()
	notificationManager := resolver.resolveNotificationManager()
	raceControl := resolver.ResolveRaceControl()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for range c {
			// ^C, handle it
			if process.IsRunning() {
				event := process.Event()

				opts, err := store.LoadServerOptions()

				if err == nil && opts.RestartEventOnServerManagerLaunch == 1 {
					// save the event so it can be started again next time server-manager starts
					err := store.UpsertLastRaceEvent(event)

					if err != nil {
						logrus.WithError(err).Error("Could not save last server event")
					}
				}

				if event.IsChampionship() && !event.IsPractice() {
					if err := championshipManager.StopActiveEvent(); err != nil {
						logrus.WithError(err).Errorf("Error stopping Championship event")
					}
				} else if event.IsRaceWeekend() && !event.IsPractice() {
					if err := raceWeekendManager.StopActiveSession(); err != nil {
						logrus.WithError(err).Errorf("Error stopping Race Weekend session")
					}
				} else {
					if err := process.Stop(); err != nil {
						logrus.WithError(err).Errorf("Could not stop server")
					}
				}
			}

			if err := notificationManager.Stop(); err != nil {
				logrus.WithError(err).Errorf("Could not stop notification manager")
			}

			raceControl.persistTimingData()

			os.Exit(0)
		}
	}()

	raceManager := resolver.resolveRaceManager()
	go panicCapture(raceManager.LoopRaces)

	err = raceManager.InitScheduledRaces()

	if err != nil {
		return err
	}

	err = championshipManager.InitScheduledChampionships()

	if err != nil {
		return err
	}

	err = raceWeekendManager.WatchForScheduledSessions()

	if err != nil {
		return err
	}

	carManager := resolver.resolveCarManager()

	go func() {
		err = carManager.CreateOrOpenSearchIndex()

		if err != nil {
			logrus.WithError(err).Error("Could not open search index")
		}
	}()

	if opts.RestartEventOnServerManagerLaunch == 1 {
		if lastEvent, err := store.LoadLastRaceEvent(); err == nil && lastEvent != nil {
			var err error

			switch a := lastEvent.(type) {
			case *ActiveChampionship:
				if !a.IsPractice() {
					err = championshipManager.applyConfigAndStart(a)
				} else {
					err = raceManager.applyConfigAndStart(a)
				}
			case *ActiveRaceWeekend:
				if !a.IsPractice() {
					err = raceWeekendManager.applyConfigAndStart(a)
				} else {
					err = raceManager.applyConfigAndStart(a)
				}
			default:
				err = raceManager.applyConfigAndStart(a)
			}

			if err != nil {
				logrus.WithError(err).Errorf("Could not start last running event")
			}
		}
	}

	return nil
}
