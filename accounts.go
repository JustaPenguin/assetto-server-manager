package servermanager

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/sethvargo/go-diceware/diceware"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/scrypt"
)

const (
	sessionUserID         = "user_id"
	requestContextKeyUser = "user"
	adminUserName         = "admin"
)

type ServerAccountOptions struct {
	IsOpen bool
}

var accountOptions = &ServerAccountOptions{
	IsOpen: false,
}

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

	Name  string
	Group Group

	PasswordHash string
	PasswordSalt string

	DefaultPassword string
}

func (a Account) NeedsPasswordReset() bool {
	return a.DefaultPassword != ""
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

var OpenUser = &Account{
	Name:  "Free Access",
	Group: GroupRead,
}

// MustLoginMiddleware determines whether a user needs to log in to access a given Group page
func MustLoginMiddleware(requiredGroup Group, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := getSession(r)

		userID, ok := sess.Values[sessionUserID].(string)

		if ok {
			user, err := raceManager.raceStore.FindAccountByID(userID)

			if err == nil {
				if user.HasGroupPrivilege(requiredGroup) {
					next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestContextKeyUser, user)))
					return
				} else {
					AddErrFlashQuick(w, r, "You do not have permission to view this page.")
					http.Redirect(w, r, "/", http.StatusFound)
					return
				}
			} else {
				logrus.WithError(err).Errorf("Could not find user for id: %d", userID)
			}
		}

		if requiredGroup == GroupRead && accountOptions.IsOpen {
			// if read is open, allow access and use a dummy user so the UI doesn't break
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestContextKeyUser, OpenUser)))
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

func UserFromRequest(r *http.Request) *Account {
	u, ok := r.Context().Value(requestContextKeyUser).(*Account)

	if !ok {
		return &Account{}
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
		err := loginUser(r, w)

		if err == ErrInvalidUsernameOrPassword {
			AddErrFlashQuick(w, r, "Invalid username or password. Check your details and try again.")
		} else if err == ErrUserNeedsPassword {
			AddFlashQuick(w, r, "Thanks for logging in. We need you to set up a permanent password for your account.")
			http.Redirect(w, r, "/accounts/new-password", http.StatusFound)
			return
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

const serverOptions = "server-account-options"

func toggleServerOpenStatusHandler(w http.ResponseWriter, r *http.Request) {
	err := raceManager.raceStore.GetMeta(serverOptions, &accountOptions)

	if err != nil && err != ErrMetaValueNotSet {
		panic(err) // @TODO
	}

	accountOptions.IsOpen = !accountOptions.IsOpen

	err = raceManager.raceStore.SetMeta(serverOptions, accountOptions)

	if err != nil {
		panic(err) // @TODO
	}

	AddFlashQuick(w, r, "Server openness successfully changed")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func newPasswordHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		password, repeatPassword := r.FormValue("Password"), r.FormValue("RepeatPassword")

		if password == repeatPassword {
			salt, err := generateSalt()

			if err != nil {
				panic(err) // @TODO
			}

			pass, err := hashPassword([]byte(password), []byte(salt))

			if err != nil {
				panic(err) // @TODO
			}

			user := UserFromRequest(r)
			user.DefaultPassword = ""
			user.PasswordSalt = salt
			user.PasswordHash = pass

			err = raceManager.raceStore.UpsertAccount(user)

			if err != nil {
				panic(err) // @TODO
			}

			AddFlashQuick(w, r, "Your password was successfully changed!")
			http.Redirect(w, r, "/", http.StatusFound)
			return
		} else {
			AddErrFlashQuick(w, r, "Your passwords must match")
		}
	}

	ViewRenderer.MustLoadTemplate(w, r, "accounts/new-password.html", nil)
}

func deleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")

	if err := raceManager.raceStore.DeleteAccount(accountID); err != nil {
		logrus.WithError(err).Errorf("Could not delete account")
		AddErrFlashQuick(w, r, "Could not delete account")
	} else {
		AddFlashQuick(w, r, "Account successfully deleted")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")

	user, err := raceManager.raceStore.FindAccountByID(accountID)

	if err != nil {
		panic(err) // @TODO
	}

	defaultPass, err := diceware.Generate(4)

	if err != nil {
		panic(err)
	}

	user.DefaultPassword = strings.Join(defaultPass, "-")
	user.PasswordSalt = ""
	user.PasswordHash = ""

	err = raceManager.raceStore.UpsertAccount(user)

	if err != nil {
		panic(err) // @TODO
	}

	AddFlashQuick(w, r, fmt.Sprintf("We have autogenerated a new password for %s, it is: %s", user.Name, user.DefaultPassword))
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

var ErrUserNeedsPassword = errors.New("servermanager: user needs to set a password")
var ErrInvalidUsernameOrPassword = errors.New("servermanager: invalid username or password")

func loginUser(r *http.Request, w http.ResponseWriter) error {
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
			if (user.NeedsPasswordReset() && password == user.DefaultPassword) ||
				(user.Name == adminUserName && config.Users.AdminPasswordOverride != "" && password == config.Users.AdminPasswordOverride) {
				// first log in of the user, direct them to a reset password form
				sess := getSession(r)
				sess.Values[sessionUserID] = user.ID.String()

				err := sess.Save(r, w)

				if err != nil {
					return err
				}

				return ErrUserNeedsPassword
			}

			passwordHash, err := hashPassword([]byte(password), []byte(user.PasswordSalt))

			if err != nil {
				return err
			}

			if subtle.ConstantTimeCompare([]byte(user.PasswordHash), []byte(passwordHash)) == 1 {
				sess := getSession(r)
				sess.Values[sessionUserID] = user.ID.String()

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
	delete(sess.Values, sessionUserID)

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
		defaultPass, err := diceware.Generate(4)

		if err != nil {
			panic(err)
		}

		account = &Account{
			DefaultPassword: strings.Join(defaultPass, "-"),
		}
	}

	if r.Method == http.MethodPost {
		username := r.FormValue("Username")
		group := r.FormValue("Group")

		if !isEditing {
			// creating new account
			account = NewAccount()
			account.DefaultPassword = r.FormValue("DefaultPassword")
		}

		account.Name = username
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
		"Accounts":         accounts,
		"ServerReadIsOpen": accountOptions.IsOpen,
	})
}

func hashPassword(password, salt []byte) (string, error) {
	pass, err := scrypt.Key(password, salt, 16384, 8, 1, 64)

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(pass), nil
}

func generateSalt() (string, error) {
	salt := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, salt)

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(salt), err
}
