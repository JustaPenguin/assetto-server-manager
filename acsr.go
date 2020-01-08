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
	"os"

	"github.com/sirupsen/logrus"
)

var acsrURL = "https://acsr.assettocorsaservers.com"

func init() {
	if acsrOverrideURL := os.Getenv("ACSR_URL"); acsrOverrideURL != "" {
		acsrURL = acsrOverrideURL
	}
}

type ACSRClient struct {
	Enabled   bool
	AccountID string
	APIKey    string
}

func NewACSRClient(accountID, apiKey string, enabled bool) *ACSRClient {
	return &ACSRClient{
		AccountID: accountID,
		APIKey:    apiKey,
		Enabled:   enabled && IsPremium == "true",
	}
}

// Sends a championship to ACSR, called OnEndSession and when a championship is created
func (a *ACSRClient) SendChampionship(championship Championship) {
	if !a.Enabled || len(championship.Events) == 0 {
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
		logrus.WithError(err).Error("acsr: couldn't JSON marshal championship")
		return
	}

	key, err := hex.DecodeString(a.APIKey)

	if err != nil {
		logrus.WithError(err).Error("acsr: api key in config is incorrect")
		return
	}

	encryptedChampionship, err := encrypt(output, key)

	if err != nil {
		logrus.Error("acsr: output encryption failed")
		return
	}

	req, err := http.NewRequest("POST", acsrURL+"/submit-result", bytes.NewBuffer(encryptedChampionship))

	if err != nil {
		logrus.Error(err)
		return
	}

	geoIP, err := geoIP()

	if err != nil {
		logrus.WithError(err).Error("acsr: couldn't get server geoIP for request")
		return
	}

	q := req.URL.Query()
	q.Add("baseurl", config.HTTP.BaseURL)
	q.Add("guid", a.AccountID)
	q.Add("geoip", geoIP.CountryName)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/json")

	_, err = http.DefaultClient.Do(req)

	if err != nil {
		logrus.Error(err)
		return
	}

	logrus.Debugf("acsr: updated championship: %s sent", championship.ID.String())
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
