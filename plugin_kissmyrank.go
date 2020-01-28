package servermanager

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

// kissmyrank handles configuration of the kissmyrank plugin
// https://www.racedepartment.com/downloads/kissmyrank-local-assetto-corsa-server-plugin.17667/

const (
	kissMyRankBaseFolderName     = "kissmyrank"
	kissMyRankConfigJSONFileName = "config.json"
)

func KissMyRankExecutablePath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(KissMyRankFolderPath(), "ac_kissmyrank-win.exe")
	}

	return filepath.Join(KissMyRankFolderPath(), "ac_kissmyrank-linux")
}

func fixKissMyRankExecutablePermissions() error {
	if runtime.GOOS == "linux" {
		return os.Chmod(KissMyRankExecutablePath(), 0755)
	}

	return nil
}

func KissMyRankFolderPath() string {
	return filepath.Join(ServerInstallPath, kissMyRankBaseFolderName)
}

func KissMyRankConfigPath() string {
	return filepath.Join(KissMyRankFolderPath(), kissMyRankConfigJSONFileName)
}

// IsKissMyRankInstalled looks in the ServerInstallPath for a "kissmyrank" directory with the correct kissmyrank executable for the given platform
func IsKissMyRankInstalled() bool {
	if _, err := os.Stat(KissMyRankExecutablePath()); os.IsNotExist(err) {
		return false
	} else if err != nil {
		logrus.WithError(err).Error("Could not determine if kissmyrank is enabled")
		return false
	} else {
		return true
	}
}

type kissMyRankConfigurationTemplateVars struct {
	BaseTemplateVars

	Form           *Form
	IsKMRInstalled bool
}

type KissMyRankHandler struct {
	*BaseHandler
	store Store
}

func NewKissMyRankHandler(baseHandler *BaseHandler, store Store) *KissMyRankHandler {
	return &KissMyRankHandler{
		BaseHandler: baseHandler,
		store:       store,
	}
}

func (kmrh *KissMyRankHandler) options(w http.ResponseWriter, r *http.Request) {
	opts, err := kmrh.store.LoadKissMyRankOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load kissmyrank options")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	form := NewForm(opts, nil, "", AccountFromRequest(r).Name == "admin")

	if r.Method == http.MethodPost {
		err := form.Submit(r)

		if err != nil {
			logrus.WithError(err).Errorf("couldn't submit form")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		err = kmrh.store.UpsertKissMyRankOptions(opts)

		if err != nil {
			logrus.WithError(err).Errorf("couldn't save KissMyRank options")
			AddErrorFlash(w, r, "Failed to save KissMyRank options")
		} else {
			AddFlash(w, r, "KissMyRank options successfully saved!")
		}
	}

	kmrh.viewRenderer.MustLoadTemplate(w, r, "server/kissmyrank-options.html", &kissMyRankConfigurationTemplateVars{
		Form:           form,
		IsKMRInstalled: IsKissMyRankInstalled(),
	})
}
