// this file was automatically generated using struct2ts github.com/JustaPenguin/assetto-server-manager.RaceControl
// +build ignore

package main

import (
	"flag"
	"io"
	"log"
	"os"

	servermanager "github.com/JustaPenguin/assetto-server-manager"
	"github.com/OneOfOne/struct2ts"
)

func main() {
	log.SetFlags(log.Lshortfile)

	var (
		out = flag.String("o", "-", "output")
		f   = os.Stdout
		err error
	)

	flag.Parse()
	if *out != "-" {
		if f, err = os.OpenFile(*out, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644); err != nil {
			panic(err)
		}
		defer f.Close()
	}
	if err = runStruct2TS(f); err != nil {
		panic(err)
	}
}

func runStruct2TS(w io.Writer) error {
	s := struct2ts.New(&struct2ts.Options{
		Indent: "    ",

		NoAssignDefaults: false,
		InterfaceOnly:    false,

		NoConstructor: false,
		NoCapitalize:  false,
		MarkOptional:  false,
		NoToObject:    false,
		NoExports:     false,
		NoHelpers:     false,
		NoDate:        false,

		ES6: false,
	})

	s.Add(servermanager.RaceControl{})

	io.WriteString(w, "// this file was automatically generated, DO NOT EDIT\n")
	return s.RenderTo(w)
}
