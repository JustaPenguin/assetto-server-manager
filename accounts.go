package servermanager

import (
	"context"
	"crypto/md5"
	"crypto/subtle"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func init() {
	// Register the Account struct with gob so that it can be stored in a session
	gob.Register(Account{})
}

func NewAccount() *Account {
	return &Account{
		ID:      uuid.New(),
		Created: time.Now(),
	}
}

type Account struct {
	ID uuid.UUID

	Created time.Time
	Updated time.Time
	Deleted time.Time

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

	if g == GroupWrite && a.Group == GroupDelete {
		return true
	}

	if g == GroupRead && (a.Group == GroupWrite || a.Group == GroupDelete) {
		return true
	}

	return false
}

type Group string

const (
	GroupRead   Group = "read"
	GroupWrite  Group = "write"
	GroupDelete Group = "delete"
	GroupAdmin  Group = "admin"
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

		if requiredGroup == GroupRead /* && config.Users.ReadOpen @TODO read open */ {
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

func DeleteAccessMiddleware(next http.Handler) http.Handler {
	return MustLoginMiddleware(GroupDelete, next)
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

func DeleteAccess(r *http.Request) func() bool {
	ok := UserFromRequest(r).HasGroupPrivilege(GroupDelete)

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

	ViewRenderer.MustLoadTemplate(w, r, "accounts/login.html", nil)
}

var ErrInvalidUsernameOrPassword = errors.New("servermanager: invalid username or password")

func LoginUser(r *http.Request, w http.ResponseWriter) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	username, password := r.FormValue("Username"), r.FormValue("Password")

	users, err := raceManager.raceStore.ListAccounts()

	if err != nil {
		return err
	}

	for _, user := range users {
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

func createOrEditAccountHandler(w http.ResponseWriter, r *http.Request) {
	var account *Account
	isEditing := false

	if id := chi.URLParam(r, "id"); id != "" {
		var err error
		account, err = raceManager.raceStore.FindAccountByID(id)

		if err != nil {
			panic(err)
		}

		isEditing = true
	} else {
		account = &Account{}
	}

	if r.Method == http.MethodPost {
		username := r.FormValue("Username")
		password := r.FormValue("Password")
		group := r.FormValue("Group")

		if !isEditing {
			// creating new account
			account = NewAccount()
		}

		account.Name = username

		if (isEditing && password != "") || !isEditing {
			account.PasswordMD5Hash = md5EncodePassword(password)
		}

		account.Group = Group(group)

		err := raceManager.raceStore.UpsertAccount(account)

		if err != nil {
			panic(err)
		}

		if isEditing {
			AddFlashQuick(w, r, "Account successfully edited")
		} else {
			AddFlashQuick(w, r, "Account successfully created")
		}

		http.Redirect(w, r, "/accounts", http.StatusFound)
		return
	}

	ViewRenderer.MustLoadTemplate(w, r, "accounts/new.html", map[string]interface{}{
		"Account":   account,
		"IsEditing": isEditing,
	})
}

func manageAccountsHandler(w http.ResponseWriter, r *http.Request) {
	accounts, err := raceManager.raceStore.ListAccounts()

	if err != nil {
		panic(err)
	}

	ViewRenderer.MustLoadTemplate(w, r, "accounts/manage.html", map[string]interface{}{
		"Accounts": accounts,
	})
}
