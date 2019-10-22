package servermanager

type Store interface {
	// Custom Races
	UpsertCustomRace(race *CustomRace) error
	FindCustomRaceByID(uuid string) (*CustomRace, error)
	ListCustomRaces() ([]*CustomRace, error)
	DeleteCustomRace(race *CustomRace) error

	// Entrants
	UpsertEntrant(entrant Entrant) error
	ListEntrants() ([]*Entrant, error)
	DeleteEntrant(id string) error

	// Servers
	ListServers() ([]*Server, error)
	FindServerByID(uuid string) (*Server, error)
	DeleteServer(uuid string) error
	UpsertServer(server *Server) error

	// Championships
	UpsertChampionship(c *Championship) error
	ListChampionships() ([]*Championship, error)
	LoadChampionship(id string) (*Championship, error)
	DeleteChampionship(id string) error

	// Live Timings
	UpsertLiveFrames([]string) error
	ListPrevFrames() ([]string, error)

	// Meta
	SetMeta(key string, value interface{}) error
	GetMeta(key string, out interface{}) error

	// Accounts
	ListAccounts() ([]*Account, error)
	UpsertAccount(a *Account) error
	FindAccountByName(name string) (*Account, error)
	FindAccountByID(id string) (*Account, error)
	DeleteAccount(id string) error

	// Audit Log
	GetAuditEntries() ([]*AuditEntry, error)
	AddAuditEntry(entry *AuditEntry) error

	// Race Weekend
	ListRaceWeekends() ([]*RaceWeekend, error)
	UpsertRaceWeekend(rw *RaceWeekend) error
	LoadRaceWeekend(id string) (*RaceWeekend, error)
	DeleteRaceWeekend(id string) error

	// Deprecated: Use the XXXServer methods below.
	//UpsertServerOptions(so *GlobalServerConfig) error

	// Deprecated: Use the XXXServer methods below.
	//LoadServerOptions() (*GlobalServerConfig, error)
}
