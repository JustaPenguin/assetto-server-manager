package main

import (
	"fmt"

	"github.com/JustaPenguin/assetto-server-manager"
)

func main() {
	data := &servermanager.Entrant{}

	formElems := servermanager.NewForm(data, nil, "", true).Fields()

	for _, formElem := range formElems {
		fmt.Println(formElem.HTML())
		fmt.Println()
	}
}
