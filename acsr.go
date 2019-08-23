package servermanager

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

// Sends a result file to ACSR, called OnEndSession
func ACSRSendResult(sessionFile string) {
	result, err := LoadResult(sessionFile + ".json")

	if err != nil {
		logrus.Error(err)
		return
	}

	output, err := json.Marshal(result)
	if err != nil {
		logrus.Error(err)
		return
	}

	_, err = http.Post(config.ACSR.URL+"/submit-result", "application/json", bytes.NewBuffer(output))

	if err != nil {
		logrus.Error(err)
		return
	}

	logrus.Debug("Dummy result files sent")
}
