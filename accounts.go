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

	"github.com/cj123/sessions"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/sethvargo/go-diceware/diceware"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/scrypt"
)

const (
	sessionAccountID            = "account_id"
	requestContextKeyAccount    = "account"
	adminUserName               = "admin"
	serverAccountOptionsMetaKey = "server-account-options"
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

var OpenAccount = &Account{
	Name:  "Free Access",
	Group: GroupRead,
}

// MustLoginMiddleware determines whether an account needs to log in to access a given Group page
func (ah *AccountHandler) MustLoginMiddleware(requiredGroup Group, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess := getSession(r)

		accountID, ok := sess.Values[sessionAccountID].(string)

		if ok {
			account, err := ah.store.FindAccountByID(accountID)

			if err == nil {
				if account.HasGroupPrivilege(requiredGroup) {
					next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestContextKeyAccount, account)))
					return
				} else {
					AddErrorFlash(w, r, "You do not have permission to view this page.")
					http.Redirect(w, r, "/", http.StatusFound)
					return
				}
			} else {
				logrus.WithError(err).Errorf("Could not find account for id: %s", accountID)
				delete(sess.Values, sessionAccountID)
				_ = sessions.Save(r, w)

				AddFlash(w, r, "You have been logged out")

				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
		}

		if requiredGroup == GroupRead && accountOptions.IsOpen {
			// if read is open, allow access and use a dummy account so the UI doesn't break
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestContextKeyAccount, OpenAccount)))
			return
		}

		if !ok {
			AddErrorFlash(w, r, "You do not have permission to view this page. Please login first.")
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
	})
}

func (ah *AccountHandler) ReadAccessMiddleware(next http.Handler) http.Handler {
	return ah.MustLoginMiddleware(GroupRead, next)
}

func (ah *AccountHandler) WriteAccessMiddleware(next http.Handler) http.Handler {
	return ah.MustLoginMiddleware(GroupWrite, next)
}

func (ah *AccountHandler) DeleteAccessMiddleware(next http.Handler) http.Handler {
	return ah.MustLoginMiddleware(GroupDelete, next)
}

func (ah *AccountHandler) AdminAccessMiddleware(next http.Handler) http.Handler {
	return ah.MustLoginMiddleware(GroupAdmin, next)
}

func AccountFromRequest(r *http.Request) *Account {
	u, ok := r.Context().Value(requestContextKeyAccount).(*Account)

	if !ok {
		return &Account{}
	}

	return u
}

func ReadAccess(r *http.Request) func() bool {
	ok := AccountFromRequest(r).HasGroupPrivilege(GroupRead)

	return func() bool {
		return ok
	}
}

func LoggedIn(r *http.Request) func() bool {
	account := AccountFromRequest(r)

	ok := account.Name != "" && account != OpenAccount

	return func() bool {
		return ok
	}
}

func WriteAccess(r *http.Request) func() bool {
	ok := AccountFromRequest(r).HasGroupPrivilege(GroupWrite)

	return func() bool {
		return ok
	}
}

func DeleteAccess(r *http.Request) func() bool {
	ok := AccountFromRequest(r).HasGroupPrivilege(GroupDelete)

	return func() bool {
		return ok
	}
}

func AdminAccess(r *http.Request) func() bool {
	ok := AccountFromRequest(r).HasGroupPrivilege(GroupAdmin)

	return func() bool {
		return ok
	}
}

type AccountHandler struct {
	*BaseHandler

	store          Store
	accountManager *AccountManager
}

func NewAccountHandler(baseHandler *BaseHandler, store Store, accountManager *AccountManager) *AccountHandler {
	return &AccountHandler{
		BaseHandler:    baseHandler,
		store:          store,
		accountManager: accountManager,
	}
}

func (ah *AccountHandler) login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := ah.accountManager.login(r, w)

		if err == ErrInvalidUsernameOrPassword {
			AddErrorFlash(w, r, "Invalid username or password. Check your details and try again.")
		} else if err == ErrAccountNeedsPassword {
			AddFlash(w, r, "Thanks for logging in. We need you to set up a permanent password for your account.")
			http.Redirect(w, r, "/accounts/new-password", http.StatusFound)
			return
		} else if err != nil {
			logrus.Errorf("Couldn't log in account, err: %s", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		} else { // err == nil, successful auth
			AddFlash(w, r, "Thanks for logging in!")
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
	}

	ah.viewRenderer.MustLoadTemplate(w, r, "accounts/login.html", nil)
}

func (ah *AccountHandler) toggleServerOpenStatus(w http.ResponseWriter, r *http.Request) {
	err := ah.store.GetMeta(serverAccountOptionsMetaKey, &accountOptions)

	if err != nil && err != ErrValueNotSet {
		logrus.WithError(err).Errorf("Could not determine server open status")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	accountOptions.IsOpen = !accountOptions.IsOpen

	err = ah.store.SetMeta(serverAccountOptionsMetaKey, accountOptions)

	if err != nil {
		logrus.WithError(err).Errorf("Could not save server open status")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	AddFlash(w, r, "Server openness successfully changed")
	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ah *AccountHandler) newPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		password, repeatPassword := r.FormValue("Password"), r.FormValue("RepeatPassword")

		if password == repeatPassword {
			account := AccountFromRequest(r)

			if err := ah.accountManager.changePassword(account, password); err == nil {
				AddFlash(w, r, "Your password was successfully changed!")
				http.Redirect(w, r, "/", http.StatusFound)
				return
			} else {
				AddErrorFlash(w, r, "Unable to change your password")
				logrus.WithError(err).Errorf("Could not change password for account id: %s", account.ID.String())
			}
		} else {
			AddErrorFlash(w, r, "Your passwords must match")
		}
	}

	ah.viewRenderer.MustLoadTemplate(w, r, "accounts/new-password.html", nil)
}

func (ah *AccountHandler) deleteAccount(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")

	if err := ah.store.DeleteAccount(accountID); err != nil {
		logrus.WithError(err).Errorf("Could not delete account")
		AddErrorFlash(w, r, "Could not delete account")
	} else {
		AddFlash(w, r, "Account successfully deleted")
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

func (ah *AccountHandler) resetPassword(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")

	account, err := ah.accountManager.resetPassword(accountID)

	if err != nil {
		AddErrorFlash(w, r, "Unable to reset account password")
		logrus.WithError(err).Errorf("Could not reset password for account id: %s", accountID)
	} else {
		AddFlash(w, r, fmt.Sprintf("We have autogenerated a new password for %s, it is: %s", account.Name, account.DefaultPassword))
	}

	http.Redirect(w, r, r.Referer(), http.StatusFound)
}

var ErrAccountNeedsPassword = errors.New("servermanager: account needs to set a password")
var ErrInvalidUsernameOrPassword = errors.New("servermanager: invalid username or password")

type AccountManager struct {
	store Store
}

func NewAccountManager(store Store) *AccountManager {
	return &AccountManager{
		store: store,
	}
}

func (am *AccountManager) login(r *http.Request, w http.ResponseWriter) error {
	if err := r.ParseForm(); err != nil {
		return err
	}

	username, password := r.FormValue("Username"), r.FormValue("Password")

	accounts, err := am.store.ListAccounts()

	if err != nil {
		return err
	}

	for _, account := range accounts {
		if username == account.Name {
			if (account.NeedsPasswordReset() && password == account.DefaultPassword) ||
				(account.Name == adminUserName && config.Accounts.AdminPasswordOverride != "" && password == config.Accounts.AdminPasswordOverride) {
				// first log in of the account, direct them to a reset password form
				sess := getSession(r)
				sess.Values[sessionAccountID] = account.ID.String()

				err := sess.Save(r, w)

				if err != nil {
					return err
				}

				return ErrAccountNeedsPassword
			}

			passwordHash, err := hashPassword([]byte(password), []byte(account.PasswordSalt))

			if err != nil {
				return err
			}

			if subtle.ConstantTimeCompare([]byte(account.PasswordHash), []byte(passwordHash)) == 1 {
				sess := getSession(r)
				sess.Values[sessionAccountID] = account.ID.String()

				return sess.Save(r, w)
			} else {
				break
			}
		}
	}

	return ErrInvalidUsernameOrPassword
}

func (am *AccountManager) resetPassword(accountID string) (*Account, error) {
	account, err := am.store.FindAccountByID(accountID)

	if err != nil {
		return nil, err
	}

	defaultPass, err := diceware.Generate(4)

	if err != nil {
		return nil, err
	}

	account.DefaultPassword = strings.Join(defaultPass, "-")
	account.PasswordSalt = ""
	account.PasswordHash = ""

	return account, am.store.UpsertAccount(account)
}

func (am *AccountManager) changePassword(account *Account, password string) error {
	salt, err := generateSalt()

	if err != nil {
		return err
	}

	pass, err := hashPassword([]byte(password), []byte(salt))

	if err != nil {
		return err
	}

	account.DefaultPassword = ""
	account.PasswordSalt = salt
	account.PasswordHash = pass

	return am.store.UpsertAccount(account)
}

func (ah *AccountHandler) logout(w http.ResponseWriter, r *http.Request) {
	sess := getSession(r)
	delete(sess.Values, sessionAccountID)

	_ = sess.Save(r, w)

	http.Redirect(w, r, "/", http.StatusFound)
}

func (ah *AccountHandler) createOrEditAccount(w http.ResponseWriter, r *http.Request) {
	var account *Account
	isEditing := false

	if id := chi.URLParam(r, "id"); id != "" {
		var err error
		account, err = ah.store.FindAccountByID(id)

		if err != nil {
			logrus.WithError(err).Errorf("Could not find account for id: %s", id)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		isEditing = true
	} else {
		defaultPass, err := diceware.Generate(4)

		if err != nil {
			logrus.WithError(err).Errorf("Could not generate password")
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
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

		err := ah.store.UpsertAccount(account)

		if err != nil {
			logrus.WithError(err).Errorf("Could save account with id: %s", account.ID)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		if isEditing {
			AddFlash(w, r, "Account successfully edited")
		} else {
			AddFlash(w, r, "Account successfully created")
		}

		http.Redirect(w, r, "/accounts", http.StatusFound)
		return
	}

	ah.viewRenderer.MustLoadTemplate(w, r, "accounts/new.html", map[string]interface{}{
		"Account":   account,
		"IsEditing": isEditing,
	})
}

func (ah *AccountHandler) manageAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := ah.store.ListAccounts()

	if err != nil {
		logrus.WithError(err).Errorf("Could not list accounts")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	ah.viewRenderer.MustLoadTemplate(w, r, "accounts/manage.html", map[string]interface{}{
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
