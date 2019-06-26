package servermanager

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"sort"
	"time"
)

type AuditEntry struct {
	UserGroup Group
	Method    string
	Url       string
	User      string
	Time      time.Time
}

var ignoredURLS = [5]string{
	"/audit-logs",
	"/quick",
	"/logs",
	"/custom",
	"/api/logs",
}

func AuditLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, url := range ignoredURLS {
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
			UserGroup: account.Group,
			Method:    r.Method,
			Url:       r.URL.String(),
			User:      account.Name,
			Time:      time.Now(),
		}

		err := raceManager.raceStore.AddAuditEntry(entry)

		if err != nil {
			logrus.WithError(err).Error("Couldn't add audit entry for request")
		}

		next.ServeHTTP(w, r)
	})
}

func serverAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	// load server audits
	auditLogs, err := raceManager.raceStore.GetAuditEntries()

	if err != nil {
		logrus.WithError(err).Error("couldn't find audit logs")
		AddErrorFlash(w, r, "Couldn't open audit logs")
	}

	// sort to newest first
	sort.Slice(auditLogs, func(i, j int) bool {
		return auditLogs[i].Time.After(auditLogs[j].Time)
	})

	// render audit log page
	ViewRenderer.MustLoadTemplate(w, r, "server/audit-logs.html", map[string]interface{}{
		"auditLogs": auditLogs,
	})
}
