package servermanager

import (
	"net/http"

	"github.com/gorilla/mux"
)

var (
	ViewRenderer *Renderer
)

func Router() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/", homeHandler)
	r.HandleFunc("/server", serverConfigHandler)

	return r
}

// homeHandler serves content to /
func homeHandler(w http.ResponseWriter, r *http.Request) {
	ViewRenderer.MustLoadTemplate(w, r, "home.html", nil)
}

func serverConfigHandler(w http.ResponseWriter, r *http.Request) {
	ViewRenderer.MustLoadTemplate(w, r, "server_config.html", map[string]interface{}{
		"config": NewForm(ConfigIniDefault.Server),
	})
}
