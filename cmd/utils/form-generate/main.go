package main

import (
	"fmt"
	"github.com/cj123/assetto-server-manager"
)

func main() {
	data := &servermanager.Entrant{}

	formElems := servermanager.NewForm(data, nil, "").Fields()

	for _, formElem := range formElems {
		fmt.Println(formElem.HTML())
		fmt.Println()
	}
}
