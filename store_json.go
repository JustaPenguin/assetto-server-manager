package servermanager

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	maxAuditEntries = 1000

	// private data
	accountsDir       = "accounts"
	serverOptionsFile = "server_options.json"
	frameLinksFile    = "frame_links.json"
	serverMetaDir     = "meta"
	auditFile         = "audit.json"

	// shared data
	championshipsDir = "championships"
	customRacesDir   = "custom_races"
	entrantsFile     = "entrants.json"
)

func NewJSONStore(dir string, sharedDir string) Store {
	return &JSONStore{
		base:   dir,
		shared: sharedDir,
	}
}

type JSONStore struct {
	base   string
	shared string

	mutex sync.RWMutex
}

func (rs *JSONStore) listFiles(dir string) ([]string, error) {
	files, err := ioutil.ReadDir(dir)

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	var list []string

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		list = append(list, strings.TrimSuffix(file.Name(), ".json"))
	}

	return list, nil
}

func (rs *JSONStore) encodeFile(path string, filename string, data interface{}) error {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	filename = filepath.Join(path, filename)

	dir := filepath.Dir(filename)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)

		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	f, err := os.Create(filename)

	if err != nil {
		return err
	}

	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")

	return enc.Encode(data)
}

func (rs *JSONStore) decodeFile(path string, filename string, out interface{}) error {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	filename = filepath.Join(path, filename)

	f, err := os.Open(filename)

	if err != nil {
		return err
	}

	defer f.Close()

	enc := json.NewDecoder(f)

	return enc.Decode(out)
}

func (rs *JSONStore) UpsertCustomRace(race *CustomRace) error {
	return rs.encodeFile(rs.shared, filepath.Join(customRacesDir, race.UUID.String()+".json"), race)
}

func (rs *JSONStore) FindCustomRaceByID(uuid string) (*CustomRace, error) {
	var customRace *CustomRace

	err := rs.decodeFile(rs.shared, filepath.Join(customRacesDir, uuid+".json"), &customRace)

	if err != nil {
		return nil, err
	}

	return customRace, nil
}

func (rs *JSONStore) ListCustomRaces() ([]*CustomRace, error) {
	files, err := rs.listFiles(filepath.Join(rs.shared, customRacesDir))

	if err != nil {
		return nil, err
	}

	var customRaces []*CustomRace

	for _, file := range files {
		race, err := rs.FindCustomRaceByID(file)

		if err != nil || !race.Deleted.IsZero() {
			continue
		}

		customRaces = append(customRaces, race)
	}

	return customRaces, nil
}

func (rs *JSONStore) DeleteCustomRace(race *CustomRace) error {
	race.Deleted = time.Now()

	return rs.UpsertCustomRace(race)
}

func (rs *JSONStore) UpsertEntrant(entrant Entrant) error {
	entrants, err := rs.ListEntrants()

	if err != nil {
		return err
	}

	entrants = append(entrants, &entrant)

	return rs.encodeFile(rs.shared, entrantsFile, entrants)
}

func (rs *JSONStore) DeleteEntrant(id string) error {
	entrants, err := rs.ListEntrants()

	if err != nil {
		return err
	}

	deleteIndex := -1

	for index, entrant := range entrants {
		if entrant.ID() == id {
			deleteIndex = index
			break
		}
	}

	if deleteIndex < 0 {
		return nil
	}

	entrants = append(entrants[:deleteIndex], entrants[deleteIndex+1:]...)

	return rs.encodeFile(rs.shared, entrantsFile, entrants)
}

func (rs *JSONStore) ListEntrants() ([]*Entrant, error) {
	var entrants []*Entrant

	err := rs.decodeFile(rs.shared, entrantsFile, &entrants)

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return entrants, nil
}

func (rs *JSONStore) UpsertServerOptions(so *GlobalServerConfig) error {
	return rs.encodeFile(rs.base, serverOptionsFile, so)
}

func (rs *JSONStore) LoadServerOptions() (*GlobalServerConfig, error) {
	var out *GlobalServerConfig

	err := rs.decodeFile(rs.base, serverOptionsFile, &out)

	if os.IsNotExist(err) {
		return &ConfigIniDefault.GlobalServerConfig, rs.UpsertServerOptions(&ConfigIniDefault.GlobalServerConfig)
	} else if err != nil {
		return nil, err
	}

	return out, err
}

func (rs *JSONStore) UpsertChampionship(c *Championship) error {
	return rs.encodeFile(rs.shared, filepath.Join(championshipsDir, c.ID.String()+".json"), c)
}

func (rs *JSONStore) ListChampionships() ([]*Championship, error) {
	files, err := rs.listFiles(filepath.Join(rs.shared, championshipsDir))

	if err != nil {
		return nil, err
	}

	var championships []*Championship

	for _, file := range files {
		c, err := rs.LoadChampionship(file)

		if err != nil || !c.Deleted.IsZero() {
			continue
		}

		championships = append(championships, c)
	}

	return championships, nil
}

func (rs *JSONStore) LoadChampionship(id string) (*Championship, error) {
	var championship *Championship

	err := rs.decodeFile(rs.shared, filepath.Join(championshipsDir, id+".json"), &championship)

	if err != nil {
		return nil, err
	}

	return championship, nil
}

func (rs *JSONStore) DeleteChampionship(id string) error {
	c, err := rs.LoadChampionship(id)

	if err != nil {
		return err
	}

	c.Deleted = time.Now()

	return rs.UpsertChampionship(c)
}

func (rs *JSONStore) UpsertLiveFrames(frameLinks []string) error {
	return rs.encodeFile(rs.base, frameLinksFile, frameLinks)
}

func (rs *JSONStore) ListPrevFrames() ([]string, error) {
	var links []string

	err := rs.decodeFile(rs.base, frameLinksFile, &links)

	if os.IsNotExist(err) {
		return links, nil
	} else if err != nil {
		return nil, err
	}

	return links, nil
}

func (rs *JSONStore) ListAccounts() ([]*Account, error) {
	files, err := rs.listFiles(filepath.Join(rs.base, accountsDir))

	if err != nil {
		return nil, err
	}

	var accounts []*Account

	for _, file := range files {
		a, err := rs.FindAccountByName(file)

		if err != nil || !a.Deleted.IsZero() {
			continue
		}

		accounts = append(accounts, a)
	}

	return accounts, nil
}

func (rs *JSONStore) UpsertAccount(a *Account) error {
	return rs.encodeFile(rs.base, filepath.Join(accountsDir, a.Name+".json"), a)
}

func (rs *JSONStore) FindAccountByName(name string) (*Account, error) {
	var account *Account

	err := rs.decodeFile(rs.base, filepath.Join(accountsDir, name+".json"), &account)

	if err != nil {
		return nil, err
	}

	return account, nil
}

func (rs *JSONStore) FindAccountByID(id string) (*Account, error) {
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

func (rs *JSONStore) DeleteAccount(id string) error {
	account, err := rs.FindAccountByID(id)

	if err != nil {
		return err
	}

	account.Deleted = time.Now()

	return rs.UpsertAccount(account)
}

func (rs *JSONStore) SetMeta(key string, value interface{}) error {
	return rs.encodeFile(rs.base, filepath.Join(serverMetaDir, key+".json"), value)
}

func (rs *JSONStore) GetMeta(key string, out interface{}) error {
	err := rs.decodeFile(rs.base, filepath.Join(serverMetaDir, key+".json"), out)

	if os.IsNotExist(err) {
		return ErrValueNotSet
	}

	return err
}

func (rs *JSONStore) GetAuditEntries() ([]*AuditEntry, error) {
	var entries []*AuditEntry

	err := rs.decodeFile(rs.base, auditFile, &entries)

	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (rs *JSONStore) AddAuditEntry(entry *AuditEntry) error {
	entries, err := rs.GetAuditEntries()

	if err != nil && !os.IsNotExist(err) {
		return err
	}

	entries = append(entries, entry)

	if len(entries) > maxAuditEntries {
		entries = entries[20:]
	}

	return rs.encodeFile(rs.base, auditFile, entries)
}
