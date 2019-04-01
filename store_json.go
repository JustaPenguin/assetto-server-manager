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
	customRacesDir    = "custom_races"
	championshipsDir  = "championships"
	accountsDir       = "accounts"
	entrantsFile      = "entrants.json"
	serverOptionsFile = "server_options.json"
	frameLinksFile    = "frame_links.json"
	serverMetaDir     = "meta"
)

func NewJSONStore(dir string) Store {
	return &JSONStore{
		base: dir,
	}
}

type JSONStore struct {
	base string

	mutex sync.RWMutex
}

func (rs *JSONStore) listFiles(dir string) ([]string, error) {
	dir = filepath.Join(rs.base, dir)

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

func (rs *JSONStore) encodeFile(filename string, data interface{}) error {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	filename = filepath.Join(rs.base, filename)

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

func (rs *JSONStore) decodeFile(filename string, out interface{}) error {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	filename = filepath.Join(rs.base, filename)

	f, err := os.Open(filename)

	if err != nil {
		return err
	}

	defer f.Close()

	enc := json.NewDecoder(f)

	return enc.Decode(out)
}

func (rs *JSONStore) UpsertCustomRace(race *CustomRace) error {
	return rs.encodeFile(filepath.Join(customRacesDir, race.UUID.String()+".json"), race)
}

func (rs *JSONStore) FindCustomRaceByID(uuid string) (*CustomRace, error) {
	var customRace *CustomRace

	err := rs.decodeFile(filepath.Join(customRacesDir, uuid+".json"), &customRace)

	if err != nil {
		return nil, err
	}

	return customRace, nil
}

func (rs *JSONStore) ListCustomRaces() ([]*CustomRace, error) {
	files, err := rs.listFiles(customRacesDir)

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

	return rs.encodeFile(entrantsFile, entrants)
}

func (rs *JSONStore) ListEntrants() ([]*Entrant, error) {
	var entrants []*Entrant

	err := rs.decodeFile(entrantsFile, &entrants)

	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return entrants, nil
}

func (rs *JSONStore) UpsertServerOptions(so *GlobalServerConfig) error {
	return rs.encodeFile(serverOptionsFile, so)
}

func (rs *JSONStore) LoadServerOptions() (*GlobalServerConfig, error) {
	var out *GlobalServerConfig

	err := rs.decodeFile(serverOptionsFile, &out)

	if os.IsNotExist(err) {
		return &ConfigIniDefault.GlobalServerConfig, rs.UpsertServerOptions(&ConfigIniDefault.GlobalServerConfig)
	} else if err != nil {
		return nil, err
	}

	return out, err
}

func (rs *JSONStore) UpsertChampionship(c *Championship) error {
	return rs.encodeFile(filepath.Join(championshipsDir, c.ID.String()+".json"), c)
}

func (rs *JSONStore) ListChampionships() ([]*Championship, error) {
	files, err := rs.listFiles(championshipsDir)

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

	err := rs.decodeFile(filepath.Join(championshipsDir, id+".json"), &championship)

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
	return rs.encodeFile(frameLinksFile, frameLinks)
}

func (rs *JSONStore) ListPrevFrames() ([]string, error) {
	var links []string

	err := rs.decodeFile(frameLinksFile, &links)

	if os.IsNotExist(err) {
		return links, nil
	} else if err != nil {
		return nil, err
	}

	return links, nil
}

func (rs *JSONStore) ListAccounts() ([]*Account, error) {
	files, err := rs.listFiles(accountsDir)

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
	return rs.encodeFile(filepath.Join(accountsDir, a.Name+".json"), a)
}

func (rs *JSONStore) FindAccountByName(name string) (*Account, error) {
	var account *Account

	err := rs.decodeFile(filepath.Join(accountsDir, name+".json"), &account)

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
	return rs.encodeFile(filepath.Join(serverMetaDir, key+".json"), value)
}

func (rs *JSONStore) GetMeta(key string, out interface{}) error {
	err := rs.decodeFile(filepath.Join(serverMetaDir, key+".json"), out)

	if os.IsNotExist(err) {
		return ErrMetaValueNotSet
	}

	return err
}
