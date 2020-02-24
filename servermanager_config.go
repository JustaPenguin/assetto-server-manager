package servermanager

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/cj123/sessions"
	"github.com/etcd-io/bbolt"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var config *Configuration

type Configuration struct {
	HTTP          HTTPConfig          `yaml:"http"`
	Steam         SteamConfig         `yaml:"steam"`
	Store         StoreConfig         `yaml:"store"`
	LiveMap       LiveMapConfig       `yaml:"live_map"`
	Server        ServerExtraConfig   `yaml:"server"`
	Accounts      AccountsConfig      `yaml:"accounts"`
	Monitoring    MonitoringConfig    `yaml:"monitoring"`
	Championships ChampionshipsConfig `yaml:"championships"`
	Lua           LuaConfig           `yaml:"lua"`
}

type ChampionshipsConfig struct {
	RecaptchaConfig struct {
		SiteKey   string `yaml:"site_key"`
		SecretKey string `yaml:"secret_key"`
	} `yaml:"recaptcha"`
}

type MonitoringConfig struct {
	Enabled bool `yaml:"enabled"`
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
	Hostname         string `yaml:"hostname"`
	SessionKey       string `yaml:"session_key"`
	SessionStoreType string `yaml:"session_store_type"`
	SessionStorePath string `yaml:"session_store_path"`
	BaseURL          string `yaml:"server_manager_base_URL"`
}

type LuaConfig struct {
	Enabled bool `yaml:"enabled"`
}

const (
	sessionStoreCookie     = "cookie"
	sessionStoreFilesystem = "filesystem"
)

func (h *HTTPConfig) createSessionStore() (sessions.Store, error) {
	switch h.SessionStoreType {
	case sessionStoreFilesystem:
		if info, err := os.Stat(h.SessionStorePath); os.IsNotExist(err) {
			err := os.MkdirAll(h.SessionStorePath, 0755)

			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		} else if !info.IsDir() {
			return nil, errors.New("servermanager: session store location must be a directory")
		}

		return sessions.NewFilesystemStore(h.SessionStorePath, []byte(h.SessionKey)), nil

	case sessionStoreCookie:
		fallthrough
	default:
		return sessions.NewCookieStore([]byte(h.SessionKey)), nil
	}
}

type SteamConfig struct {
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	InstallPath    string `yaml:"install_path"`
	ForceUpdate    bool   `yaml:"force_update"`
	ExecutablePath string `yaml:"executable_path"`
}

type StoreConfig struct {
	Type       string `yaml:"type"`
	Path       string `yaml:"path"`
	SharedPath string `yaml:"shared_data_path"`
}

func (s *StoreConfig) BuildStore() (Store, error) {
	var rs Store

	if s.SharedPath == "" {
		s.SharedPath = s.Path
	}

	switch s.Type {
	case "boltdb":
		bbdb, err := bbolt.Open(s.Path, 0644, nil)

		if err != nil {
			return nil, err
		}

		rs = NewBoltStore(bbdb)
	case "json":
		rs = NewJSONStore(s.Path, s.SharedPath)
	default:
		return nil, fmt.Errorf("invalid store type (%s), must be either boltdb/json", s.Type)
	}

	if err := Migrate(rs); err != nil {
		return nil, err
	}

	return rs, nil
}

type ServerExtraConfig struct {
	Plugins                     []*CommandPlugin `yaml:"plugins"`
	AuditLogging                bool             `yaml:"audit_logging"`
	PerformanceMode             bool             `yaml:"performance_mode"`
	DisableWindowsBrowserOpen   bool             `yaml:"dont_open_browser"`
	ScanContentFolderForChanges bool             `yaml:"scan_content_folder_for_changes"`

	// Deprecated: use Plugins instead
	RunOnStart []string `yaml:"run_on_start"`
}

type CommandPlugin struct {
	Executable string   `yaml:"executable"`
	Arguments  []string `yaml:"arguments"`
}

func (c *CommandPlugin) String() string {
	out := c.Executable
	out += strings.Join(c.Arguments, " ")

	return out
}

func ReadConfig(location string) (conf *Configuration, err error) {
	f, err := os.Open(location)

	if err != nil {
		return nil, err
	}

	defer f.Close()

	if err := yaml.NewDecoder(f).Decode(&conf); err != nil {
		return nil, err
	}

	config = conf
	sessionsStore, err = conf.HTTP.createSessionStore()

	if err != nil {
		return nil, err
	}

	if config.Accounts.AdminPasswordOverride != "" {
		logrus.Infof("WARNING! Admin Password Override is set. Please only have this set if you are resetting your admin account password!")
	}

	if config.Steam.ExecutablePath == "" {
		config.Steam.ExecutablePath = ServerExecutablePath
	}

	return conf, err
}

type Theme string

const (
	ThemeDefault = "default"
	ThemeLight   = "light"
	ThemeDark    = "dark"
)

type ThemeDetails struct {
	Theme Theme
	Name  string
}

var ThemeOptions = []ThemeDetails{
	{
		Theme: ThemeDefault,
		Name:  "Use Default",
	},
	{
		Theme: ThemeLight,
		Name:  "Light",
	},
	{
		Theme: ThemeDark,
		Name:  "Dark",
	},
}
