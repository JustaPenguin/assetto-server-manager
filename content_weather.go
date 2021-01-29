package servermanager

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cj123/ini"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

const weatherInfoFile = "weather.ini"

type Weather map[string]string

func ListWeather() (Weather, error) {
	// defaultWeather is loaded if there aren't any weather options on the server
	defaultWeather := Weather{
		"1_heavy_fog":    "Heavy Fog",
		"2_light_fog":    "Light Fog",
		"3_clear":        "Clear",
		"4_mid_clear":    "Mid Clear",
		"5_light_clouds": "Light Clouds",
		"6_mid_clouds":   "Mid Clouds",
		"7_heavy_clouds": "Heavy Clouds",
	}

	baseDir := filepath.Join(ServerInstallPath, "content", "weather")

	weatherFolders, err := ioutil.ReadDir(baseDir)

	if os.IsNotExist(err) {
		return defaultWeather, nil
	} else if err != nil {
		return nil, err
	}

	weather := defaultWeather

	for _, weatherFolder := range weatherFolders {
		if !weatherFolder.IsDir() {
			continue
		}

		// read the weather info file
		name, err := getWeatherName(baseDir, weatherFolder.Name())

		if err != nil {
			return nil, err
		}

		if name == "" {
			name = weatherFolder.Name()
		}

		weather[weatherFolder.Name()] = name
	}

	if len(weather) == 0 {
		return defaultWeather, nil
	}

	return weather, nil
}

func getWeatherName(folder, weather string) (string, error) {
	f, err := os.Open(filepath.Join(folder, weather, weatherInfoFile))

	if err != nil {
		return "", nil
	}

	defer f.Close()

	i, err := ini.Load(f)

	if err != nil {
		return "", err
	}

	s, err := i.GetSection("LAUNCHER")

	if err != nil {
		return "", err
	}

	k, err := s.GetKey("NAME")

	if err != nil {
		return "", nil
	}

	return k.String(), nil
}

func getWeatherType(weather string) (int, error) {
	baseDir := filepath.Join(ServerInstallPath, "content", "weather")

	f, err := os.Open(filepath.Join(baseDir, weather, weatherInfoFile))

	if err != nil {
		return 0, err
	}

	defer f.Close()

	i, err := ini.Load(f)

	if err != nil {
		return 0, err
	}

	s, err := i.GetSection("__LAUNCHER_CM")

	if err != nil {
		return 0, err
	}

	k, err := s.GetKey("WEATHER_TYPE")

	if err != nil {
		return 0, nil
	}

	weatherType, err := k.Int()

	if err != nil {
		return 0, nil
	}

	return weatherType, nil
}

type WeatherHandler struct {
	*BaseHandler
}

func NewWeatherHandler(baseHandler *BaseHandler) *WeatherHandler {
	return &WeatherHandler{
		BaseHandler: baseHandler,
	}
}

type weatherListTemplateVars struct {
	BaseTemplateVars

	Weathers Weather
}

func (wh *WeatherHandler) list(w http.ResponseWriter, r *http.Request) {
	weather, err := ListWeather()

	if err != nil {
		logrus.WithError(err).Errorf("could not get weather list")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	wh.viewRenderer.MustLoadTemplate(w, r, "content/weather.html", &weatherListTemplateVars{
		Weathers: weather,
	})
}

func (wh *WeatherHandler) delete(w http.ResponseWriter, r *http.Request) {
	weatherKey := chi.URLParam(r, "key")
	weatherPath := filepath.Join(ServerInstallPath, "content", "weather")

	existingWeather, err := ListWeather()

	if err != nil {
		logrus.WithError(err).Errorf("could not get weather list")
		AddErrorFlash(w, r, "Couldn't get weather list")
		http.Redirect(w, r, r.Referer(), http.StatusFound)
		return
	}

	var found bool

	for key := range existingWeather {
		if weatherKey == key {
			// Delete car
			found = true

			err := os.RemoveAll(filepath.Join(weatherPath, weatherKey))

			if err != nil {
				found = false
				logrus.WithError(err).Errorf("could not remove weather files")
			}

			delete(existingWeather, key)
			break
		}
	}

	if found {
		// confirm deletion
		AddFlash(w, r, "Weather preset successfully deleted!")
	} else {
		// inform weather wasn't found
		AddErrorFlash(w, r, "Sorry, weather preset could not be deleted. Are you sure it was installed?")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}
