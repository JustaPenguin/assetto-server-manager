package values

import (
	"log"
	"strings"
)

var _ = log.Print

type CSV []string

func (csv *CSV) EncodeICalValue() (string, error) {
	return strings.Join(*csv, ","), nil
}

func (csv *CSV) DecodeICalValue(value string) error {
	value = strings.TrimSpace(value)
	*csv = CSV(strings.Split(value, ","))
	return nil
}

func NewCSV(items ...string) *CSV {
	return (*CSV)(&items)
}
