package servermanager

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxAuditEntries = 1000

	// private data
	accountsDir           = "accounts"
	serverOptionsFile     = "server_options.json"
	frameLinksFile        = "frame_links.json"
	serverMetaDir         = "meta"
	auditFile             = "audit.json"
	strackerOptionsFile   = "stracker_options.json"
	kissMyRankOptionsFile = "kissmyrank_options.json"
	liveTimingsDataFile   = "live_timings.json"
	lastRaceEventFile     = "last_race_event.json"

	// shared data
	championshipsDir = "championships"
	raceWeekendsDir  = "race_weekends"
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

func (rs *JSONStore) writeFile(path, filename string, data []byte) error {
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

	_, err = f.Write(data)
	return err
}

func (rs *JSONStore) deleteFile(path, filename string) error {
	rs.mutex.Lock()
	defer rs.mutex.Unlock()

	return os.Remove(filepath.Join(path, filename))
}

func (rs *JSONStore) readFile(path, filename string) ([]byte, error) {
	rs.mutex.RLock()
	defer rs.mutex.RUnlock()

	filename = filepath.Join(path, filename)

	f, err := os.Open(filename)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	return ioutil.ReadAll(f)
}

func (rs *JSONStore) encodeFile(path, filename string, data interface{}) error {
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

func (rs *JSONStore) decodeFile(path, filename string, out interface{}) error {
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
	race.Updated = time.Now()

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

	sort.Slice(customRaces, func(i, j int) bool {
		return customRaces[i].Updated.After(customRaces[j].Updated)
	})

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

	isNew := true

	for _, newEntrant := range entrants {
		if newEntrant.GUID == entrant.GUID {
			newEntrant = &entrant
			isNew = false

			break
		}
	}

	if isNew {
		entrants = append(entrants, &entrant)
	}

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
		defaultConfig := ConfigIniDefault()

		return &defaultConfig.GlobalServerConfig, rs.UpsertServerOptions(&defaultConfig.GlobalServerConfig)
	} else if err != nil {
		return nil, err
	}

	return out, err
}

func (rs *JSONStore) UpsertChampionship(c *Championship) error {
	c.Updated = time.Now()

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

	sort.Slice(championships, func(i, j int) bool {
		return championships[i].Updated.After(championships[j].Updated)
	})

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
	files, err := rs.listFiles(filepath.Join(rs.shared, accountsDir))

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
	a.Updated = time.Now()

	return rs.encodeFile(rs.shared, filepath.Join(accountsDir, a.Name+".json"), a)
}

func (rs *JSONStore) FindAccountByName(name string) (*Account, error) {
	var account *Account

	err := rs.decodeFile(rs.shared, filepath.Join(accountsDir, name+".json"), &account)

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

func (rs *JSONStore) ListRaceWeekends() ([]*RaceWeekend, error) {
	files, err := rs.listFiles(filepath.Join(rs.shared, raceWeekendsDir))

	if err != nil {
		return nil, err
	}

	var raceWeekends []*RaceWeekend

	for _, file := range files {
		rw, err := rs.LoadRaceWeekend(file)

		if err != nil || !rw.Deleted.IsZero() {
			continue
		}

		raceWeekends = append(raceWeekends, rw)
	}

	sort.Slice(raceWeekends, func(i, j int) bool {
		return raceWeekends[i].Updated.After(raceWeekends[j].Updated)
	})

	return raceWeekends, nil
}

func (rs *JSONStore) UpsertRaceWeekend(rw *RaceWeekend) error {
	rw.Updated = time.Now()

	return rs.encodeFile(rs.shared, filepath.Join(raceWeekendsDir, rw.ID.String()+".json"), rw)
}

func (rs *JSONStore) LoadRaceWeekend(id string) (*RaceWeekend, error) {
	var raceWeekend *RaceWeekend

	err := rs.decodeFile(rs.shared, filepath.Join(raceWeekendsDir, id+".json"), &raceWeekend)

	if os.IsNotExist(err) {
		return nil, ErrRaceWeekendNotFound
	} else if err != nil {
		return nil, err
	}

	return raceWeekend, nil
}

func (rs *JSONStore) DeleteRaceWeekend(id string) error {
	rw, err := rs.LoadRaceWeekend(id)

	if err != nil {
		return err
	}

	rw.Deleted = time.Now()

	return rs.UpsertRaceWeekend(rw)
}

func (rs *JSONStore) UpsertStrackerOptions(sto *StrackerConfiguration) error {
	return rs.encodeFile(rs.base, strackerOptionsFile, sto)
}

func (rs *JSONStore) LoadStrackerOptions() (*StrackerConfiguration, error) {
	var out *StrackerConfiguration

	err := rs.decodeFile(rs.base, strackerOptionsFile, &out)

	if os.IsNotExist(err) {
		strackerConfig := DefaultStrackerIni()

		return strackerConfig, rs.UpsertStrackerOptions(strackerConfig)
	} else if err != nil {
		return nil, err
	}

	return out, err
}

func (rs *JSONStore) UpsertKissMyRankOptions(kmr *KissMyRankConfig) error {
	return rs.encodeFile(rs.base, kissMyRankOptionsFile, kmr)
}

func (rs *JSONStore) LoadKissMyRankOptions() (*KissMyRankConfig, error) {
	var out *KissMyRankConfig

	err := rs.decodeFile(rs.base, kissMyRankOptionsFile, &out)

	if os.IsNotExist(err) {
		kmrConfig := DefaultKissMyRankConfig()

		return kmrConfig, rs.UpsertKissMyRankOptions(kmrConfig)
	} else if err != nil {
		return nil, err
	}

	return out, err
}

func (rs *JSONStore) UpsertLiveTimingsData(lt *LiveTimingsPersistedData) error {
	return rs.encodeFile(rs.base, liveTimingsDataFile, lt)
}

func (rs *JSONStore) LoadLiveTimingsData() (*LiveTimingsPersistedData, error) {
	var lt *LiveTimingsPersistedData

	err := rs.decodeFile(rs.base, liveTimingsDataFile, &lt)

	if err != nil {
		return nil, err
	}

	return lt, err
}

func (rs *JSONStore) UpsertLastRaceEvent(r RaceEvent) error {
	raceEvent, err := marshalRaceEvent(r)

	if err != nil {
		return err
	}

	return rs.writeFile(rs.base, lastRaceEventFile, raceEvent)
}

func (rs *JSONStore) LoadLastRaceEvent() (RaceEvent, error) {
	data, err := rs.readFile(rs.base, lastRaceEventFile)

	if err != nil {
		return nil, err
	}

	return unmarshalRaceEvent(data)
}

func (rs *JSONStore) ClearLastRaceEvent() error {
	err := rs.deleteFile(rs.base, lastRaceEventFile)

	if err != nil && os.IsNotExist(err) {
		return nil
	}

	return err
}
