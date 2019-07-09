package servermanager

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/etcd-io/bbolt"
)

type BoltStore struct {
	db *bbolt.DB
}

func NewBoltStore(db *bbolt.DB) Store {
	return &BoltStore{db: db}
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

func (rs *BoltStore) customRaceBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(customRaceBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(customRaceBucketName)
}

func (rs *BoltStore) encode(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

func (rs *BoltStore) decode(data []byte, out interface{}) error {
	return json.Unmarshal(data, out)
}

func (rs *BoltStore) UpsertCustomRace(race *CustomRace) error {
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

func (rs *BoltStore) FindCustomRaceByID(uuid string) (*CustomRace, error) {
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

func (rs *BoltStore) ListCustomRaces() ([]*CustomRace, error) {
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

func (rs *BoltStore) DeleteCustomRace(race *CustomRace) error {
	race.Deleted = time.Now()

	return rs.UpsertCustomRace(race)
}

func (rs *BoltStore) entrantsBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(entrantsBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(entrantsBucketName)
}

func (rs *BoltStore) UpsertEntrant(entrant Entrant) error {
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

func (rs *BoltStore) DeleteEntrant(id string) error {
	return rs.db.Update(func(tx *bbolt.Tx) error {
		bkt, err := rs.entrantsBucket(tx)

		if err != nil {
			return err
		}

		return bkt.Delete([]byte(id))
	})
}

func (rs *BoltStore) ListEntrants() ([]*Entrant, error) {
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

func (rs *BoltStore) frameLinksBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(frameLinksBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(frameLinksBucketName)
}

func (rs *BoltStore) UpsertLiveFrames(frameLinks []string) error {
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

func (rs *BoltStore) ListPrevFrames() ([]string, error) {
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

func (rs *BoltStore) serverOptionsBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(serverOptionsBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(serverOptionsBucketName)
}

func (rs *BoltStore) UpsertServerOptions(so *GlobalServerConfig) error {
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

func (rs *BoltStore) LoadServerOptions() (*GlobalServerConfig, error) {
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

func (rs *BoltStore) championshipsBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(championshipsBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(championshipsBucketName)
}

func (rs *BoltStore) UpsertChampionship(c *Championship) error {
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

func (rs *BoltStore) ListChampionships() ([]*Championship, error) {
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

func (rs *BoltStore) LoadChampionship(id string) (*Championship, error) {
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

func (rs *BoltStore) DeleteChampionship(id string) error {
	championship, err := rs.LoadChampionship(id)

	if err != nil {
		return err
	}

	championship.Deleted = time.Now()

	return rs.UpsertChampionship(championship)
}

func (rs *BoltStore) accountsBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(accountsBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(accountsBucketName)
}

func (rs *BoltStore) ListAccounts() ([]*Account, error) {
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

func (rs *BoltStore) UpsertAccount(a *Account) error {
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

func (rs *BoltStore) FindAccountByName(name string) (*Account, error) {
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

func (rs *BoltStore) FindAccountByID(id string) (*Account, error) {
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

func (rs *BoltStore) DeleteAccount(id string) error {
	account, err := rs.FindAccountByID(id)

	if err != nil {
		return err
	}

	account.Deleted = time.Now()

	return rs.UpsertAccount(account)
}

var metaBucketName = []byte("meta")

func (rs *BoltStore) metaBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(metaBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(metaBucketName)
}

func (rs *BoltStore) SetMeta(key string, value interface{}) error {
	return rs.db.Update(func(tx *bbolt.Tx) error {
		bkt, err := rs.metaBucket(tx)

		if err != nil {
			return err
		}

		enc, err := rs.encode(value)

		if err != nil {
			return err
		}

		return bkt.Put([]byte(key), enc)
	})
}

var ErrValueNotSet = errors.New("servermanager: value not set")

func (rs *BoltStore) GetMeta(key string, out interface{}) error {
	err := rs.db.View(func(tx *bbolt.Tx) error {
		bkt, err := rs.metaBucket(tx)

		if err == bbolt.ErrBucketNotFound {
			return ErrValueNotSet
		} else if err != nil {
			return err
		}

		val := bkt.Get([]byte(key))

		if val == nil {
			return ErrValueNotSet
		}

		err = rs.decode(val, &out)

		return err
	})

	return err
}

var auditBucketName = []byte("audit")

func (rs *BoltStore) auditBucket(tx *bbolt.Tx) (*bbolt.Bucket, error) {
	if !tx.Writable() {
		bkt := tx.Bucket(auditBucketName)

		if bkt == nil {
			return nil, bbolt.ErrBucketNotFound
		}

		return bkt, nil
	}

	return tx.CreateBucketIfNotExists(auditBucketName)
}

func (rs *BoltStore) GetAuditEntries() ([]*AuditEntry, error) {
	var audits []*AuditEntry

	err := rs.db.View(func(tx *bbolt.Tx) error {
		bkt, err := rs.auditBucket(tx)

		if err == bbolt.ErrBucketNotFound {
			return ErrValueNotSet
		} else if err != nil {
			return err
		}

		val := bkt.Get([]byte("audit"))

		if val == nil {
			return ErrValueNotSet
		}

		err = rs.decode(val, &audits)

		return err
	})

	return audits, err
}

func (rs *BoltStore) AddAuditEntry(entry *AuditEntry) error {
	entries, err := rs.GetAuditEntries()

	if err != nil && err != ErrValueNotSet {
		return err
	}

	entries = append(entries, entry)

	if len(entries) > maxAuditEntries {
		entries = entries[20:]
	}

	return rs.db.Update(func(tx *bbolt.Tx) error {
		bkt, err := rs.auditBucket(tx)

		if err != nil {
			return err
		}

		enc, err := rs.encode(entries)

		if err != nil {
			return err
		}

		return bkt.Put([]byte("audit"), enc)
	})
}

func (rs *BoltStore) DoPreMigration() error {
	//only done with JSON database
	return nil
}
