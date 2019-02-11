package main

import (
	"github.com/cj123/assetto-server-manager/pkg/acd"
	"os"
)

func main() {
	f, err := os.Open("data.acd")

	if err != nil {
		panic(err)
	}

	defer f.Close()

	_, err = acd.NewReader(f)

	if err != nil {
		panic(err)
	}
}
