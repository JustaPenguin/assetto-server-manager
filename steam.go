package servermanager

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"github.com/solovev/steam_go"
)

type SteamLoginHandler struct{}

func (slh *SteamLoginHandler) redirectToSteamLogin(backURLFunc func(r *http.Request) string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		opId := steam_go.NewOpenId(r)
		switch opId.Mode() {
		case "":
			http.Redirect(w, r, opId.AuthUrl(), 301)
		case "cancel":
			logrus.Error("Steam authorization cancelled")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		default:
			steamID, err := opId.ValidateAndGetId()

			if err != nil {
				logrus.WithError(err).Error("Could not validate steamID")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, backURLFunc(r)+"?steamGUID="+steamID, http.StatusFound)
		}
	}
}
