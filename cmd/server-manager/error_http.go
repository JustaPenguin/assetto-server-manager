package main

import (
	"fmt"
	"net"
	"net/http"
	"runtime"
	"strings"

	"github.com/pkg/browser"
	"github.com/sirupsen/logrus"
)

type HTTPErrorHandler struct {
	Cause string
	Error error
}

const httpErrorMessage = `!!! An Error Occurred !!!
-------------------------

Failed to initialise server manager. 

Your configuration file is probably incorrect, or you haven't followed the instructions in the README properly. 
Please carefully check that your options are set correctly.

      Error Details
-------------------------

The error occurred attempting to: %s
The error more specifically is: %s

-------------------------

If you need support, you can ask in the RaceDepartment Support thread. Be sure to copy the above errors into your post.
	
	Support thread: https://www.racedepartment.com/threads/ac-server-manager.165662/

We're really active on the support thread helping everybody with their problems and adding new features.

Please don't post a negative review if you get stuck on this screen. We worked really hard on server manager and
we want people to see an accurate representation of the awesome tool we know server manager is.

- The Emperor Servers Team

Email: support@emperorservers.com
GitHub: https://github.com/JustaPenguin/assetto-server-manager

`

func (h *HTTPErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, httpErrorMessage, h.Cause, h.Error)
}

func (h *HTTPErrorHandler) String() string {
	return fmt.Sprintf(httpErrorMessage, h.Cause, h.Error)
}

func ServeHTTPWithError(addr string, cause string, err error) {
	h := &HTTPErrorHandler{Cause: cause, Error: err}

	fmt.Println(h.String())

	listener, err := net.Listen("tcp", addr)

	if err != nil {
		return
	}

	if runtime.GOOS == "windows" {
		_ = browser.OpenURL("http://" + strings.Replace(addr, "0.0.0.0", "127.0.0.1", 1))
	}

	if err := http.Serve(listener, h); err != nil {
		logrus.Fatal(err)
	}
}
