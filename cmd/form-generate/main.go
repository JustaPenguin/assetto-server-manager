package main

import (
	"fmt"
	"github.com/cj123/assetto-server-manager"
)

func main() {
	data := &servermanager.ConfigIniDefault

	formElems := servermanager.NewForm(data, nil, "quick").Fields()

	for _, formElem := range formElems {
		fmt.Println(formElem.HTML())
		fmt.Println()
	}
}
