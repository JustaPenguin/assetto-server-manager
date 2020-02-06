package servermanager

import (
	"net/http"
	"sort"
	"time"

	"github.com/sirupsen/logrus"
)

type AuditEntry struct {
	UserGroup Group
	Method    string
	URL       string
	User      string
	Time      time.Time
}

var ignoredURLs = [5]string{
	"/audit-logs",
	"/quick",
	"/logs",
	"/custom",
	"/api/logs",
}

type AuditLogHandler struct {
	*BaseHandler

	store Store
}

func NewAuditLogHandler(baseHandler *BaseHandler, store Store) *AuditLogHandler {
	return &AuditLogHandler{
		BaseHandler: baseHandler,
		store:       store,
	}
}

func (alh *AuditLogHandler) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, url := range ignoredURLs {
			if url == r.URL.String() {
				next.ServeHTTP(w, r)
				return
			}
		}

		account := AccountFromRequest(r)

		if account == nil {
			next.ServeHTTP(w, r)
			return
		}

		entry := &AuditEntry{
			UserGroup: account.Group(),
			Method:    r.Method,
			URL:       r.URL.String(),
			User:      account.Name,
			Time:      time.Now(),
		}

		err := alh.store.AddAuditEntry(entry)

		if err != nil {
			logrus.WithError(err).Error("Couldn't add audit entry for request")
		}

		next.ServeHTTP(w, r)
	})
}

type auditLogTemplateVars struct {
	BaseTemplateVars

	AuditLogs []*AuditEntry
}

func (alh *AuditLogHandler) viewLogs(w http.ResponseWriter, r *http.Request) {
	// load server audits
	auditLogs, err := alh.store.GetAuditEntries()

	if err != nil {
		logrus.WithError(err).Error("couldn't find audit logs")
		AddErrorFlash(w, r, "Couldn't open audit logs")
	}

	// sort to newest first
	sort.Slice(auditLogs, func(i, j int) bool {
		return auditLogs[i].Time.After(auditLogs[j].Time)
	})

	// render audit log page
	alh.viewRenderer.MustLoadTemplate(w, r, "server/audit-logs.html", &auditLogTemplateVars{
		AuditLogs: auditLogs,
	})
}
