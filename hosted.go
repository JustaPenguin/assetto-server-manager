package servermanager

import (
	"os"
)

var (
	IsHosted           = os.Getenv("HOSTED") == "true"
	MaxClientsOverride = formValueAsInt(os.Getenv("MAX_CLIENTS_OVERRIDE"))
	IsPremium          = "true"
)

func Premium() bool {
	return IsPremium == "true"
}
