package servermanager

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/sirupsen/logrus"
)

var acsrURL = "https://acsr.assettocorsaservers.com"

func init() {
	gob.Register(Championship{})

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
		Enabled:   enabled && Premium(),
	}
}

// Sends a championship to ACSR, called OnEndSession and when a championship is created
func (a *ACSRClient) SendChampionship(inChampionship Championship) {
	if !a.Enabled || len(inChampionship.Events) == 0 {
		return
	}

	if !baseURLIsSet() {
		logrus.Errorf("Cannot send Championship to ACSR - no baseURL is set.")
		return
	}

	if !baseURLIsValid() {
		logrus.Errorf("Cannot send Championship to ACSR - baseURL is not valid.")
		return
	}

	// championships are cloned before being sent to ACSR. this prevents any issues with pointers within the
	// struct being erroneously modified in our original championship struct.
	championship, err := cloneChampionship(inChampionship)

	if err != nil {
		logrus.WithError(err).Errorf("Cannot clone Championship for ACSR")
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

	geoIP, err := geoIP()

	if err != nil {
		logrus.WithError(err).Error("Could not get GeoIP data for server")
		return
	}

	resp, err := a.send("/submit-result", championship, map[string]string{
		"baseurl": config.HTTP.BaseURL,
		"geoip":   geoIP.CountryName,
	})

	if err != nil {
		logrus.WithError(err).Error("could not submit championship to ACSR")
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode < 400 {
		logrus.Debugf("acsr: updated championship: %s sent", championship.ID.String())
	} else {
		logrus.Errorf("acsr: sent championship: %s was not accepted. (status: %d) Please check your credentials.", championship.ID.String(), resp.StatusCode)
	}
}

func (a *ACSRClient) send(url string, data interface{}, queryParams map[string]string) (*http.Response, error) {
	output, err := json.Marshal(data)

	if err != nil {
		return nil, err
	}

	key, err := hex.DecodeString(a.APIKey)

	if err != nil {
		return nil, err
	}

	encryptedData, err := encrypt(output, key)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", acsrURL+url, bytes.NewBuffer(encryptedData))

	if err != nil {
		return nil, err
	}

	q := req.URL.Query()

	for key, val := range queryParams {
		q.Add(key, val)
	}

	q.Add("guid", a.AccountID)
	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/json")

	return http.DefaultClient.Do(req)
}

type ACSRDriverRatingRequest struct {
	GUIDs []string `json:"guids"`
}

type ACSRDriverRating struct {
	DriverID         uint    `json:"driver_id"`
	SkillRatingGrade string  `json:"skill_rating_grade"`
	SkillRating      float64 `json:"skill_rating"`
	SafetyRating     int     `json:"safety_rating"`
	NumEvents        int     `json:"num_events"`
	IsProvisional    bool    `json:"is_provisional"`
}

func (a *ACSRClient) GetRating(guids ...string) (map[string]*ACSRDriverRating, error) {
	data := ACSRDriverRatingRequest{}

	anonymisedGUIDs := make(map[string]string)

	for _, guid := range guids {
		anonymised := AnonymiseDriverGUID(guid)
		anonymisedGUIDs[anonymised] = guid
		data.GUIDs = append(data.GUIDs, anonymised)
	}

	resp, err := a.send("/api/ratings", data, nil)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("servermanager: acsr request responded with a bad status code (%d). check your credentials", resp.StatusCode)
	}

	var anonymisedOut map[string]*ACSRDriverRating

	if err := json.NewDecoder(resp.Body).Decode(&anonymisedOut); err != nil {
		return nil, err
	}

	normalGUIDMap := make(map[string]*ACSRDriverRating)

	for anonymisedGUID, data := range anonymisedOut {
		if guid, ok := anonymisedGUIDs[anonymisedGUID]; ok {
			normalGUIDMap[guid] = data
		}
	}

	return normalGUIDMap, nil
}

// cloneChampionship takes a Championship and returns a complete new copy of it.
func cloneChampionship(c Championship) (out Championship, err error) {
	buf := new(bytes.Buffer)

	err = gob.NewEncoder(buf).Encode(c)

	if err != nil {
		return out, err
	}

	err = gob.NewDecoder(buf).Decode(&out)

	return out, err
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

func baseURLIsValid() bool {
	if !baseURLIsSet() {
		return false
	}

	_, err := url.Parse(config.HTTP.BaseURL)

	return err == nil
}

func baseURLIsSet() bool {
	return config != nil && config.HTTP.BaseURL != ""
}
