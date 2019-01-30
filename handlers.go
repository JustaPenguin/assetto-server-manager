package servermanager

import (
	"github.com/sirupsen/logrus"
	"net/http"

	"github.com/gorilla/mux"
)

var (
	ViewRenderer *Renderer
)

func Router() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/server-options", globalServerOptionsHandler)
	r.HandleFunc("/race-options", raceOptionsHandler)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static", http.FileServer(http.Dir("./static"))))

	return r
}

// homeHandler serves content to /
func homeHandler(w http.ResponseWriter, r *http.Request) {
	ViewRenderer.MustLoadTemplate(w, r, "home.html", nil)
}

func globalServerOptionsHandler(w http.ResponseWriter, r *http.Request) {
	form := NewForm(&ConfigIniDefault.Server.GlobalServerConfig, nil)

	if r.Method == http.MethodPost {
		err := form.Submit(r)

		if err != nil {
			logrus.Errorf("couldn't submit form, err: %s", err)
		}

		// save the config
		err = ConfigIniDefault.Write()

		if err != nil {
			logrus.Errorf("couldn't save config, err: %s", err)
		}
	}

	ViewRenderer.MustLoadTemplate(w, r, "global_server_options.html", map[string]interface{}{
		"form": form,
	})
}

func raceOptionsHandler(w http.ResponseWriter, r *http.Request) {
	var dataMap = make(map[string][]string)

	var carNames []string
	cars, err := ListCars()

	if err != nil {
		logrus.Fatalf("could not get car list, err: %s", err)
	}

	for _, car := range cars {
		carNames = append(carNames, car.Name)
	}

	dataMap["CarOpts"] = carNames

	form := NewForm(&ConfigIniDefault.Server.CurrentRaceConfig, dataMap)

	if r.Method == http.MethodPost {
		err := form.Submit(r)

		if err != nil {
			logrus.Errorf("couldn't submit form, err: %s", err)
		}

		// save the config
		err = ConfigIniDefault.Write()

		if err != nil {
			logrus.Errorf("couldn't save config, err: %s", err)
		}
	}

	ViewRenderer.MustLoadTemplate(w, r, "current_race_options.html", map[string]interface{}{
		"form": form,
	})
}
