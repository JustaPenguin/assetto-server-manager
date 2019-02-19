package servermanager

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"net/http"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

func init() {
	// Register the Account struct with gob so that it can be stored in a session
	gob.Register(Account{})
}

type Account struct {
	Name            string `yaml:"name"`
	Group           Group  `yaml:"group"`
	PasswordMD5Hash string `yaml:"password"`
}

func (a Account) HasGroupPrivilege(g Group) bool {
	if g == a.Group {
		return true
	}

	if a.Group == GroupAdmin {
		return true
	}

	if g == GroupRead && a.Group == GroupWrite {
		return true
	}

	return false
}

type Group string

const (
	GroupRead  Group = "read"
	GroupWrite Group = "write"
	GroupAdmin Group = "admin"
)

var OpenUser = Account{
	Name:            "Free Access",
	Group:           GroupRead,
	PasswordMD5Hash: "",
}

// MustLoginMiddleware determines whether a user needs to log in to access a given Group page
func MustLoginMiddleware(requiredGroup Group, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := getSession(r)

		user, ok := sess.Values["user"].(Account)

		if ok && user.HasGroupPrivilege(requiredGroup) {
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "user", user)))
			return
		} else if ok {
			AddErrFlashQuick(w, r, "You do not have permission to view this page.")
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		if requiredGroup == GroupRead && config.Users.ReadOpen {
			// if read is open, allow access and use a dummy user so the UI doesn't break
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "user", OpenUser)))
			return
		}

		if !ok {
			AddErrFlashQuick(w, r, "You do not have permission to view this page. Please login first.")
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
	})
}

func ReadAccessMiddleware(next http.Handler) http.Handler {
	return MustLoginMiddleware(GroupRead, next)
}

func WriteAccessMiddleware(next http.Handler) http.Handler {
	return MustLoginMiddleware(GroupWrite, next)
}

func AdminAccessMiddleware(next http.Handler) http.Handler {
	return MustLoginMiddleware(GroupAdmin, next)
}

func md5EncodePassword(plaintext string) string {
	h := md5.New()
	h.Write([]byte(plaintext))

	return hex.EncodeToString(h.Sum(nil))
}

func UserFromRequest(r *http.Request) Account {
	u, ok := r.Context().Value("user").(Account)

	if !ok {
		return Account{}
	}

	return u
}

func ReadAccess(r *http.Request) func() bool {
	ok := UserFromRequest(r).HasGroupPrivilege(GroupRead)

	return func() bool {
		return ok
	}
}

func LoggedIn(r *http.Request) func() bool {
	user := UserFromRequest(r)

	ok := user.Name != "" && user != OpenUser

	return func() bool {
		return ok
	}
}

func WriteAccess(r *http.Request) func() bool {
	ok := UserFromRequest(r).HasGroupPrivilege(GroupWrite)

	return func() bool {
		return ok
	}
}

func AdminAccess(r *http.Request) func() bool {
	ok := UserFromRequest(r).HasGroupPrivilege(GroupAdmin)

	return func() bool {
		return ok
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := LoginUser(r, w)

		if err == ErrInvalidUsernameOrPassword {
			AddErrFlashQuick(w, r, "Invalid username or password. Check your details and try again.")
		} else if err != nil {
			logrus.Errorf("Couldn't log in user, err: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else { // err == nil, successful auth
			AddFlashQuick(w, r, "Thanks for logging in!")
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
	}

	ViewRenderer.MustLoadTemplate(w, r, filepath.Join("accounts", "login.html"), nil)
}

var ErrInvalidUsernameOrPassword = errors.New("servermanager: invalid username or password")

func LoginUser(r *http.Request, w http.ResponseWriter) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	username, password := r.FormValue("Username"), r.FormValue("Password")

	for _, user := range config.Users.Accounts {
		if username == user.Name {
			if subtle.ConstantTimeCompare([]byte(user.PasswordMD5Hash), []byte(md5EncodePassword(password))) == 1 {
				sess := getSession(r)
				sess.Values["user"] = user

				return sess.Save(r, w)
			} else {
				break
			}
		}
	}

	return ErrInvalidUsernameOrPassword
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	sess := getSession(r)
	delete(sess.Values, "user")

	_ = sess.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}
