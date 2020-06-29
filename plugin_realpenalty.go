package servermanager

import (
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

// handles configuration of the realpenalty plugin
// https://www.racedepartment.com/downloads/real-penalty-tool.29591/

const (
	realPenaltyBaseFolderName = "realpenalty"
)

func RealPenaltyExecutablePath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(RealPenaltyFolderPath(), "ac_penalty.exe")
	}

	return filepath.Join(RealPenaltyFolderPath(), "ac_penalty")
}

func fixRealPenaltyExecutablePermissions() error {
	if runtime.GOOS == "linux" {
		return os.Chmod(RealPenaltyExecutablePath(), 0755)
	}

	return nil
}

func RealPenaltyFolderPath() string {
	return filepath.Join(ServerInstallPath, realPenaltyBaseFolderName)
}

func IsRealPenaltyInstalled() bool {
	if _, err := os.Stat(RealPenaltyExecutablePath()); os.IsNotExist(err) {
		return false
	} else if err != nil {
		logrus.WithError(err).Error("Could not determine if realpenalty is enabled")
		return false
	} else {
		return true
	}
}

type RealPenaltyHandler struct {
	*BaseHandler

	store Store
}

func NewRealPenaltyHandler(baseHandler *BaseHandler, store Store) *RealPenaltyHandler {
	return &RealPenaltyHandler{BaseHandler: baseHandler, store: store}
}

type realPenaltyConfigurationTemplateVars struct {
	BaseTemplateVars

	Form                   template.HTML
	IsRealPenaltyInstalled bool
}

func (rph *RealPenaltyHandler) options(w http.ResponseWriter, r *http.Request) {
	realPenaltyOptions, err := rph.store.LoadRealPenaltyOptions()

	if err != nil {
		logrus.WithError(err).Errorf("couldn't load real penalty options")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodPost {
		err := DecodeFormData(realPenaltyOptions, r)

		if err != nil {
			logrus.WithError(err).Errorf("couldn't submit form")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		err = rph.store.UpsertRealPenaltyOptions(realPenaltyOptions)

		if err != nil {
			logrus.WithError(err).Errorf("couldn't save Real Penalty options")
			AddErrorFlash(w, r, "Failed to save Real Penalty options")
		} else {
			AddFlash(w, r, "Real Penalty options successfully saved!")
		}
	}

	form, err := EncodeFormData(realPenaltyOptions, r)

	if err != nil {
		logrus.WithError(err).Errorf("Couldn't encode form data")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rph.viewRenderer.MustLoadTemplate(w, r, "server/realpenalty-options.html", &realPenaltyConfigurationTemplateVars{
		Form:                   form,
		IsRealPenaltyInstalled: IsRealPenaltyInstalled(),
	})
}
