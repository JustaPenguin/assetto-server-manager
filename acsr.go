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
func ACSRSendResult(championship *Championship) {
	if config == nil && (config.ACSR.APIKey == "" || config.ACSR.AccountID == "") {
		return
	}

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

	gcm, nonce, err := encryptChampionship()

	if err != nil {
		logrus.Error("ACSR output encryption failed")
		return
	}

	client := http.Client{}

	req, err := http.NewRequest("POST", config.ACSR.URL+"/submit-result", bytes.NewBuffer(gcm.Seal(nonce, nonce, output, nil)))

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

func encryptChampionship() (cipher.AEAD, []byte, error) {
	// encryption
	key, err := hex.DecodeString(config.ACSR.APIKey)

	if err != nil {
		logrus.WithError(err).Error("api key in config is incorrect")
		return nil, nil, err
	}

	// generate a new aes cipher using our 32 byte long key
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	// gcm or Galois/Counter Mode, is a mode of operation
	// for symmetric key cryptographic block ciphers
	// - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	gcm, err := cipher.NewGCM(c)
	// if any error generating new GCM
	// handle them
	if err != nil {
		return nil, nil, err
	}

	// creates a new byte array the size of the nonce
	// which must be passed to Seal
	nonce := make([]byte, gcm.NonceSize())
	// populates our nonce with a cryptographically secure
	// random sequence
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	return gcm, nonce, nil
}
