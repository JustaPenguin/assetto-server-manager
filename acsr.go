package servermanager

import (
	"bytes"
	"encoding/json"
	"github.com/sirupsen/logrus"
	"net/http"
)

// Sends a result file to ACSR, called OnEndSession
func ACSRSendResult(sessionFile string) {
	result, err := LoadResult(sessionFile + ".json")

	if err != nil {
		logrus.Fatal(err)
	}

	output, err := json.Marshal(result)
	if err != nil {
		return
	}

	_, err = http.Post(config.ACSR.URL+"/submit-result", "application/json",
		bytes.NewBuffer(output))

	if err != nil {
		logrus.Fatal(err)
	}

	logrus.Debug("Dummy result files sent")
}
