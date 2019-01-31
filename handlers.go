package servermanager

import (
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

var (
	ViewRenderer *Renderer
	store        = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
)

func Router() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/cars", carsHandler)
	r.HandleFunc("/tracks", tracksHandler)
	r.HandleFunc("/track/delete/{name}", trackDeleteHandler)
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

	var carNames, trackNames []string
	cars, err := ListCars()

	if err != nil {
		logrus.Fatalf("could not get car list, err: %s", err)
	}

	tracks, err := ListTracks()

	if err != nil {
		logrus.Fatalf("could not get track list, err: %s", err)
	}

	for _, car := range cars {
		carNames = append(carNames, car.Name)
	}

	for _, track := range tracks {
		trackNames = append(trackNames, track.Name)
	}

	dataMap["CarOpts"] = carNames
	dataMap["TrackOpts"] = trackNames

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

func carsHandler(w http.ResponseWriter, r *http.Request) {
	cars, err := ListCars()

	if err != nil {
		logrus.Fatalf("could not get car list, err: %s", err)
	}

	ViewRenderer.MustLoadTemplate(w, r, "cars.html", map[string]interface{}{
		"cars": cars,
	})
}

func tracksHandler(w http.ResponseWriter, r *http.Request) {
	tracks, err := ListTracks()

	if err != nil {
		logrus.Fatalf("could not get track list, err: %s", err)
	}

	ViewRenderer.MustLoadTemplate(w, r, "tracks.html", map[string]interface{}{
		"tracks": tracks,
	})
}

func trackDeleteHandler(w http.ResponseWriter, r *http.Request) {
	trackName := mux.Vars(r)["name"]
	tracksPath := filepath.Join(ServerInstallPath, "content", "tracks")

	existingTracks, err := ioutil.ReadDir(tracksPath)

	if err != nil {
		logrus.Fatalf("could not get track list, err: %s", err)
	}

	var found bool

	for _, track := range existingTracks {
		if track.Name() == trackName {
			// Delete track
			found = true

			err := os.RemoveAll(filepath.Join(tracksPath, trackName))

			if err != nil {
				found = false
			}

			break
		}
	}

	session, err := getSession(r)

	if err != nil {
		logrus.Fatalf("could not get session, err: %s", err)
	}

	if found {
		// send flash, confirm deletion
		session.AddFlash("Track successfully deleted!")
	} else {
		// send flash, inform track wasn't found
		session.AddFlash("Sorry, track could not be deleted.")
	}

	err = session.Save(r, w)

	if err != nil {
		logrus.Fatalf("could not save session, err: %s", err)
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func getSession(r *http.Request) (*sessions.Session, error) {
	session, err := store.Get(r, "messages")

	if err != nil {
		return nil, err
	}

	return session, nil
}
