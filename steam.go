package servermanager

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/solovev/steam_go"
)

type SteamLoginHandler struct{}

func (slh *SteamLoginHandler) redirectToSteamLogin(backURLFunc func(r *http.Request) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		opID := steam_go.NewOpenId(r)
		switch opID.Mode() {
		case "":
			http.Redirect(w, r, opID.AuthUrl(), 301)
		case "cancel":
			logrus.Error("Steam authorization cancelled")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		default:
			steamID, err := opID.ValidateAndGetId()

			if err != nil {
				logrus.WithError(err).Error("Could not validate steamID")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, backURLFunc(r)+"?steamGUID="+steamID, http.StatusFound)
		}
	}
}
