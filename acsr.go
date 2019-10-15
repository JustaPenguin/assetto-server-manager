package servermanager

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

// Sends a championship to ACSR, called OnEndSession and when a championship is created
func ACSRSendResult(championship Championship) {
	if config == nil || (config.ACSR.APIKey == "" || config.ACSR.AccountID == "" || !config.ACSR.Enabled) || len(championship.Events) == 0 {
		return
	}

	championship.Events = ExtractRaceWeekendSessionsIntoIndividualEvents(championship.Events)

	for _, event := range championship.Events {
		for _, session := range event.Sessions {
			if session.Completed() {
				session.Results.Anonymize()
			}
		}
	}

	output, err := json.Marshal(championship)
	if err != nil {
		logrus.WithError(err).Error("couldn't JSON marshal championship")
		return
	}

	key, err := hex.DecodeString(config.ACSR.APIKey)

	if err != nil {
		logrus.WithError(err).Error("api key in config is incorrect")
		return
	}

	encryptedChampionship, err := encrypt(output, key)

	if err != nil {
		logrus.Error("ACSR output encryption failed")
		return
	}

	client := http.Client{}

	req, err := http.NewRequest("POST", config.ACSR.URL+"/submit-result", bytes.NewBuffer(encryptedChampionship))

	if err != nil {
		logrus.Error(err)
		return
	}

	q := req.URL.Query()
	q.Add("baseurl", config.HTTP.BaseURL)
	q.Add("guid", config.ACSR.AccountID)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/json")

	_, err = client.Do(req)

	if err != nil {
		logrus.Error(err)
		return
	}

	logrus.Debugf("updated championship: %s sent to ACSR", championship.ID.String())
}

func encrypt(data, key []byte) ([]byte, error) {
	c, err := aes.NewCipher(key)

	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)

	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())

	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}
