package servermanager

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/etcd-io/bbolt"
)

type BoltRaceStore struct {
	db *bbolt.DB
}

func NewBoltRaceStore(db *bbolt.DB) RaceStore {
	return &BoltRaceStore{db: db}
}

var (
	customRaceBucketName    = []byte("customRaces")
	serverOptionsBucketName = []byte("serverOptions")
	entrantsBucketName      = []byte("entrants")
	championshipsBucketName = []byte("championships")
	accountsBucketName      = []byte("accounts")
	frameLinksBucketName    = []byte("frameLinks")

	serverOptionsKey = []byte("serverOptions")
)

func (rs *BoltRaceStore) customRaceBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(customRaceBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(customRaceBucketName)
}

func (rs *BoltRaceStore) encode(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func (rs *BoltRaceStore) decode(data []byte, out interface{}) error {
	return json.Unmarshal(data, out)
}

func (rs *BoltRaceStore) UpsertCustomRace(race *CustomRace) error {
	return rs.db.Update(func(tx *bbolt.Tx) error {
		bkt, err := rs.customRaceBucket(tx)

		if err != nil {
			return err
		}

		race.Updated = time.Now()

		encoded, err := rs.encode(race)

		if err != nil {
			return err
		}

		return bkt.Put([]byte(race.UUID.String()), encoded)
	})
}

func (rs *BoltRaceStore) FindCustomRaceByID(uuid string) (*CustomRace, error) {
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

func (rs *BoltRaceStore) ListCustomRaces() ([]*CustomRace, error) {
	var customRaces []*CustomRace

	err := rs.db.View(func(tx *bbolt.Tx) error {
		bkt, err := rs.customRaceBucket(tx)

		if err == bbolt.ErrBucketNotFound {
			return nil
		} else if err != nil {
			return err
		}

		return bkt.ForEach(func(k, v []byte) error {
			var race *CustomRace

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

func (rs *BoltRaceStore) DeleteCustomRace(race *CustomRace) error {
	race.Deleted = time.Now()

	return rs.UpsertCustomRace(race)
}

func (rs *BoltRaceStore) entrantsBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(entrantsBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(entrantsBucketName)
}

func (rs *BoltRaceStore) UpsertEntrant(entrant Entrant) error {
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

func (rs *BoltRaceStore) ListEntrants() ([]*Entrant, error) {
	var entrants []*Entrant

	err := rs.db.View(func(tx *bbolt.Tx) error {
		bkt, err := rs.entrantsBucket(tx)

		if err == bbolt.ErrBucketNotFound {
			return nil
		} else if err != nil {
			return err
		}

		return bkt.ForEach(func(k, v []byte) error {
			var entrant *Entrant

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

func (rs *BoltRaceStore) frameLinksBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(frameLinksBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(frameLinksBucketName)
}

func (rs *BoltRaceStore) UpsertLiveFrames(frameLinks []string) error {
	return rs.db.Update(func(tx *bbolt.Tx) error {
		bkt, err := rs.frameLinksBucket(tx)

		if err != nil {
			return err
		}

		encoded, err := rs.encode(frameLinks)

		if err != nil {
			return err
		}

		return bkt.Put([]byte("frameLinks"), encoded)
	})
}

func (rs *BoltRaceStore) ListPrevFrames() ([]string, error) {
	var links []string

	err := rs.db.View(func(tx *bbolt.Tx) error {
		bkt, err := rs.frameLinksBucket(tx)

		if err == bbolt.ErrBucketNotFound {
			return nil
		} else if err != nil {
			return err
		}

		linksByte := bkt.Get([]byte("frameLinks"))

		err = rs.decode(linksByte, &links)

		if err != nil {
			return err
		}

		return nil
	})

	return links, err
}

func (rs *BoltRaceStore) serverOptionsBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(serverOptionsBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(serverOptionsBucketName)
}

func (rs *BoltRaceStore) UpsertServerOptions(so *GlobalServerConfig) error {
	return rs.db.Update(func(tx *bbolt.Tx) error {
		bkt, err := rs.serverOptionsBucket(tx)

		if err != nil {
			return err
		}

		encoded, err := rs.encode(so)

		if err != nil {
			return err
		}

		return bkt.Put(serverOptionsKey, encoded)
	})
}

func (rs *BoltRaceStore) LoadServerOptions() (*GlobalServerConfig, error) {
	// start with defaults
	so := &ConfigIniDefault.GlobalServerConfig

	err := rs.db.View(func(tx *bbolt.Tx) error {
		bkt, err := rs.serverOptionsBucket(tx)

		if err != nil {
			return err
		}

		data := bkt.Get(serverOptionsKey)

		if data == nil {
			return nil
		}

		return rs.decode(data, &so)
	})

	if err == bbolt.ErrBucketNotFound {
		// no server options created yet, apply defaults
		return so, rs.UpsertServerOptions(so)
	}

	return so, err
}

func (rs *BoltRaceStore) championshipsBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(championshipsBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(championshipsBucketName)
}

func (rs *BoltRaceStore) UpsertChampionship(c *Championship) error {
	c.Updated = time.Now()

	return rs.db.Update(func(tx *bbolt.Tx) error {
		b, err := rs.championshipsBucket(tx)

		if err != nil {
			return err
		}

		data, err := rs.encode(c)

		if err != nil {
			return err
		}

		return b.Put([]byte(c.ID.String()), data)
	})
}

func (rs *BoltRaceStore) ListChampionships() ([]*Championship, error) {
	var championships []*Championship

	err := rs.db.View(func(tx *bbolt.Tx) error {
		b, err := rs.championshipsBucket(tx)

		if err == bbolt.ErrBucketNotFound {
			return nil
		} else if err != nil {
			return err
		}

		return b.ForEach(func(k, v []byte) error {
			var championship *Championship

			err := rs.decode(v, &championship)

			if err != nil {
				return err
			}

			if !championship.Deleted.IsZero() {
				// championship deleted
				return nil // continue
			}

			championships = append(championships, championship)

			return nil
		})
	})

	return championships, err
}

var ErrChampionshipNotFound = errors.New("servermanager: championship not found")

func (rs *BoltRaceStore) LoadChampionship(id string) (*Championship, error) {
	var championship *Championship

	err := rs.db.View(func(tx *bbolt.Tx) error {
		b, err := rs.championshipsBucket(tx)

		if err != nil {
			return err
		}

		data := b.Get([]byte(id))

		if data == nil {
			return ErrChampionshipNotFound
		}

		return rs.decode(data, &championship)
	})

	if err != nil {
		return nil, err
	}

	return championship, err
}

func (rs *BoltRaceStore) DeleteChampionship(id string) error {
	championship, err := rs.LoadChampionship(id)

	if err != nil {
		return err
	}

	championship.Deleted = time.Now()

	return rs.UpsertChampionship(championship)
}

func (rs *BoltRaceStore) accountsBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(accountsBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(accountsBucketName)
}

func (rs *BoltRaceStore) ListAccounts() ([]*Account, error) {
	var accounts []*Account

	err := rs.db.View(func(tx *bbolt.Tx) error {
		b, err := rs.accountsBucket(tx)

		if err == bbolt.ErrBucketNotFound {
			return nil
		} else if err != nil {
			return err
		}

		return b.ForEach(func(k, v []byte) error {
			var account *Account

			err := rs.decode(v, &account)

			if err != nil {
				return err
			}

			if !account.Deleted.IsZero() {
				// account deleted
				return nil // continue
			}

			accounts = append(accounts, account)

			return nil
		})
	})

	return accounts, err
}

func (rs *BoltRaceStore) UpsertAccount(a *Account) error {
	a.Updated = time.Now()

	return rs.db.Update(func(tx *bbolt.Tx) error {
		b, err := rs.accountsBucket(tx)

		if err != nil {
			return err
		}

		data, err := rs.encode(a)

		if err != nil {
			return err
		}

		return b.Put([]byte(a.Name), data)
	})
}

var ErrAccountNotFound = errors.New("servermanager: account not found")

func (rs *BoltRaceStore) FindAccountByName(name string) (*Account, error) {
	var account *Account

	err := rs.db.View(func(tx *bbolt.Tx) error {
		b, err := rs.accountsBucket(tx)

		if err != nil {
			return err
		}

		data := b.Get([]byte(name))

		if data == nil {
			return ErrAccountNotFound
		}

		return rs.decode(data, &account)
	})

	if err != nil {
		return nil, err
	}

	return account, err
}

func (rs *BoltRaceStore) FindAccountByID(id string) (*Account, error) {
	accounts, err := rs.ListAccounts()

	if err != nil {
		return nil, err
	}

	for _, a := range accounts {
		if a.ID.String() == id {
			return a, nil
		}
	}

	return nil, ErrAccountNotFound
}
