package main

import (
	"encoding/json"
	"fmt"
	"github.com/cj123/assetto-server-manager/pkg/udp"
	"github.com/davecgh/go-spew/spew"
	"os"
	"time"
)

var outFileName = time.Now().Format(time.ANSIC + ".json")

func main() {

	udpServerConn, err := udp.NewServerClient("127.0.0.1", 12000, 11000, false, "", callback)

	if err != nil {
		panic(err)
	}

	defer udpServerConn.Close()

	ch := make(chan struct{})

	<-ch // wait forever
}

type Entry struct {
	Received  time.Time
	EventType udp.Event

	Data udp.Message
}

var entries []Entry

func callback(message udp.Message) {
	spew.Dump(message)

	entries = append(entries, Entry{
		Received:  time.Now(),
		EventType: message.Event(),
		Data:      message,
	})

	f, err := os.Create(outFileName)

	if err != nil {
		panic(err)
	}

	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")

	err = encoder.Encode(entries)

	if err != nil {
		fmt.Println("err encoding", err)
	}
}
