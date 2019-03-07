package main

import (
	"fmt"
	"log"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"
	"github.com/cj123/assetto-server-manager/pkg/udp/replay"
)

var outFileName = time.Now().Format("2006-01-02_15.04.json")

func main() {
	callback := replay.RecordUDPMessages(outFileName)

	udpServerConn, err := udp.NewServerClient("127.0.0.1", 12002, 11002, false, "", 0, func(response udp.Message) {
		log.Println(response.Event())
		callback(response)
	})

	if err != nil {
		panic(err)
	}

	defer udpServerConn.Close()

	ch := make(chan struct{})

	<-ch // wait forever
}

func callback(message udp.Message) {
	fmt.Println(message)
}
