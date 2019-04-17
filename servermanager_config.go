package servermanager

import (
	"fmt"
	"os"

	"github.com/etcd-io/bbolt"
	"github.com/gorilla/sessions"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var config *Configuration

type Configuration struct {
	HTTP     HTTPConfig        `yaml:"http"`
	Steam    SteamConfig       `yaml:"steam"`
	Store    StoreConfig       `yaml:"store"`
	LiveMap  LiveMapConfig     `yaml:"live_map"`
	Server   ServerExtraConfig `yaml:"server"`
	Accounts AccountsConfig    `yaml:"accounts"`
}

type AccountsConfig struct {
	AdminPasswordOverride string `yaml:"admin_password_override"`
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
	BaseURL    string `yaml:"server_manager_base_URL"`
}

type SteamConfig struct {
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	InstallPath    string `yaml:"install_path"`
	ForceUpdate    bool   `yaml:"force_update"`
	ExecutablePath string `yaml:"executable_path"`
}

type StoreConfig struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
}

func (s *StoreConfig) BuildStore() (Store, error) {
	var rs Store

	switch s.Type {
	case "boltdb":
		bbdb, err := bbolt.Open(s.Path, 0644, nil)

		if err != nil {
			return nil, err
		}

		rs = NewBoltStore(bbdb)
	case "json":
		rs = NewJSONStore(s.Path)
	default:
		return nil, fmt.Errorf("invalid store type (%s), must be either boltdb/json", s.Type)
	}

	if err := Migrate(rs); err != nil {
		return nil, err
	}

	return rs, nil
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

	if config.Accounts.AdminPasswordOverride != "" {
		logrus.Infof("WARNING! Admin Password Override is set. Please only have this set if you are resetting your admin account password!")
	}

	if config.Steam.ExecutablePath == "" {
		config.Steam.ExecutablePath = serverExecutablePath
	}

	return conf, err
}
