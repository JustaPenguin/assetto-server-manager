package servermanager

import (
	"fmt"
	"io"
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
			http.Redirect(w, r, opID.AuthUrl(), http.StatusPermanentRedirect)
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

			clientSideRedirect(backURLFunc(r)+"?steamGUID="+steamID, w)
		}
	}
}

const clientSideRedirectHTML = `
<!doctype html>
<html>    
<head>      
<title>Redirect</title>      
<meta http-equiv="refresh" content="0;URL='%s'" />    
</head>    
<body> 
<p>If you are not redirected automatically, please <a href="%s">click here</a>.</p> 
</body>  
</html>     
`

func clientSideRedirect(url string, w io.Writer) {
	_, _ = w.Write([]byte(fmt.Sprintf(clientSideRedirectHTML, url, url)))
}
