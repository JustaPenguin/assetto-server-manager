package servermanager

import (
	"archive/zip"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

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

	Form                        template.HTML
	IsRealPenaltyInstalled      bool
	RealPenaltySupportedVersion string
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
		Form:                        form,
		IsRealPenaltyInstalled:      IsRealPenaltyInstalled(),
		RealPenaltySupportedVersion: RealPenaltySupportedVersion,
	})
}

func (rph *RealPenaltyHandler) downloadLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment;filename="realpenalty_logs_%s.zip"`, time.Now().Format("2006-01-02_15_04")))
	w.Header().Set("Content-Type", "application/zip")

	if err := rph.buildLogZip(w); err != nil {
		logrus.WithError(err).Errorf("Could not create real penalty log zip")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
}

func (rph *RealPenaltyHandler) buildLogZip(w io.Writer) error {
	z := zip.NewWriter(w)
	defer z.Close()

	logFiles, err := ioutil.ReadDir(filepath.Join(RealPenaltyFolderPath(), "logs"))

	if err != nil {
		return err
	}

	for _, file := range logFiles {
		if err := rph.writeRealPenaltyLogFileToZip(z, file); err != nil {
			return err
		}
	}

	return nil
}

func (rph *RealPenaltyHandler) writeRealPenaltyLogFileToZip(z *zip.Writer, info os.FileInfo) error {
	zf, err := z.Create(info.Name())

	if err != nil {
		return err
	}

	f, err := os.Open(filepath.Join(filepath.Join(RealPenaltyFolderPath(), "logs"), info.Name()))

	if err != nil {
		return err
	}

	defer f.Close()

	_, err = io.Copy(zf, f)

	return err
}
