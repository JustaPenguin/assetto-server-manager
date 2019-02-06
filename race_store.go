package servermanager

import (
	"encoding/json"
	"github.com/etcd-io/bbolt"
	"time"
)

type RaceStore struct {
	db *bbolt.DB
}

func NewRaceStore(db *bbolt.DB) *RaceStore {
	return &RaceStore{db: db}
}

var (
	customRaceBucketName = []byte("customRaces")
	entrantsBucketName   = []byte("entrants")
)

func (rs *RaceStore) customRaceBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(customRaceBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(customRaceBucketName)
}

func (rs *RaceStore) encode(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func (rs *RaceStore) decode(data []byte, out interface{}) error {
	return json.Unmarshal(data, out)
}

func (rs *RaceStore) UpsertCustomRace(race CustomRace) error {
	return rs.db.Update(func(tx *bbolt.Tx) error {
		bkt, err := rs.customRaceBucket(tx)

		if err != nil {
			return err
		}

		encoded, err := rs.encode(race)

		if err != nil {
			return err
		}

		return bkt.Put([]byte(race.UUID.String()), encoded)
	})
}

func (rs *RaceStore) FindCustomRaceByID(uuid string) (*CustomRace, error) {
	var customRace *CustomRace

	err := rs.db.View(func(tx *bbolt.Tx) error {
		bkt, err := rs.customRaceBucket(tx)

		if err != nil {
			return err
		}

		data := bkt.Get([]byte(uuid))

		if data == nil {
			return ErrCustomRaceNotFound
		}

		return rs.decode(data, &customRace)
	})

	return customRace, err
}

func (rs *RaceStore) ListCustomRaces() ([]CustomRace, error) {
	var customRaces []CustomRace

	err := rs.db.View(func(tx *bbolt.Tx) error {
		bkt, err := rs.customRaceBucket(tx)

		if err == bbolt.ErrBucketNotFound {
			return nil
		} else if err != nil {
			return err
		}

		return bkt.ForEach(func(k, v []byte) error {
			var race CustomRace

			err := rs.decode(v, &race)

			if err != nil {
				return err
			}

			if !race.Deleted.IsZero() {
				// soft deleted race, move on
				return nil
			}

			customRaces = append(customRaces, race)

			return nil
		})
	})

	return customRaces, err
}

func (rs *RaceStore) DeleteCustomRace(race CustomRace) error {
	race.Deleted = time.Now()

	return rs.UpsertCustomRace(race)
}

func (rs *RaceStore) entrantsBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(entrantsBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(entrantsBucketName)
}

func (rs *RaceStore) UpsertEntrant(entrant Entrant) error {
	return rs.db.Update(func(tx *bbolt.Tx) error {
		bkt, err := rs.entrantsBucket(tx)

		if err != nil {
			return err
		}

		// clear out some race specific values
		entrant.Model = ""
		entrant.Skin = ""
		entrant.SpectatorMode = 0

		encoded, err := rs.encode(entrant)

		if err != nil {
			return err
		}

		return bkt.Put([]byte(entrant.ID()), encoded)
	})
}

func (rs *RaceStore) ListEntrants() ([]Entrant, error) {
	var entrants []Entrant

	err := rs.db.View(func(tx *bbolt.Tx) error {
		bkt, err := rs.entrantsBucket(tx)

		if err == bbolt.ErrBucketNotFound {
			return nil
		} else if err != nil {
			return err
		}

		return bkt.ForEach(func(k, v []byte) error {
			var entrant Entrant

			err := rs.decode(v, &entrant)

			if err != nil {
				return err
			}

			entrants = append(entrants, entrant)

			return nil
		})
	})

	return entrants, err
}
