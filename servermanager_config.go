package servermanager

import (
	"fmt"
	"os"

	"github.com/etcd-io/bbolt"
	"github.com/gorilla/sessions"
	"gopkg.in/yaml.v2"
)

var config *Configuration

type Configuration struct {
	HTTP    HTTPConfig        `yaml:"http"`
	Steam   SteamConfig       `yaml:"steam"`
	Store   StoreConfig       `yaml:"store"`
	Users   UsersConfig       `yaml:"users"`
	LiveMap LiveMapConfig     `yaml:"live_map"`
	Server  ServerExtraConfig `yaml:"server"`
}

type LiveMapConfig struct {
	IntervalMs int `yaml:"refresh_interval_ms"`
}

func (l *LiveMapConfig) IsEnabled() bool {
	return l.IntervalMs > 0
}

type HTTPConfig struct {
	Hostname   string `yaml:"hostname"`
	SessionKey string `yaml:"session_key"`
}

type SteamConfig struct {
	Username    string `yaml:"username"`
	Password    string `yaml:"password"`
	InstallPath string `yaml:"install_path"`
	ForceUpdate bool   `yaml:"force_update"`
}

type StoreConfig struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

func (s *StoreConfig) BuildStore() (RaceStore, error) {
	var rs RaceStore

	switch s.Type {
	case "boltdb":
		bbdb, err := bbolt.Open(s.Path, 0644, nil)

		if err != nil {
			return nil, err
		}

		rs = NewBoltRaceStore(bbdb)
	case "json":
		rs = NewJSONRaceStore(s.Path)
	default:
		return nil, fmt.Errorf("invalid store type (%s), must be either boltdb/json", s.Type)
	}

	if err := Migrate(rs); err != nil {
		return nil, err
	}

	return rs, nil
}

type UsersConfig struct {
	Accounts []Account `yaml:"accounts"`

	ReadOpen bool `yaml:"read_open"`
}

type ServerExtraConfig struct {
	RunOnStart []string `yaml:"run_on_start"`
}

func ReadConfig(location string) (conf *Configuration, err error) {
	f, err := os.Open(location)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	err = yaml.NewDecoder(f).Decode(&conf)

	config = conf
	store = sessions.NewCookieStore([]byte(conf.HTTP.SessionKey))

	return conf, err
}
