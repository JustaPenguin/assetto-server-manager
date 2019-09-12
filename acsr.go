package servermanager

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

// Sends a championship to ACSR, called OnEndSession and when a championship is created
func ACSRSendResult(championship *Championship) {
	for _, event := range championship.Events {
		for _, session := range event.Sessions {
			if session.Completed() {
				session.Results.Anonymize()
			}
		}
	}

	output, err := json.Marshal(championship)
	if err != nil {
		logrus.Error(err)
		return
	}

	client := http.Client{}

	req, err := http.NewRequest("POST", config.ACSR.URL+"/submit-result", bytes.NewBuffer(output))

	if err != nil {
		logrus.Error(err)
		return
	}

	q := req.URL.Query()
	q.Add("baseurl", config.HTTP.BaseURL)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/json")

	_, err = client.Do(req)

	if err != nil {
		logrus.Error(err)
		return
	}

	logrus.Debug("updated championship sent to ACSR")
}
