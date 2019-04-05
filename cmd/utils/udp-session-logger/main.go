package main

import (
	"github.com/cj123/assetto-server-manager/pkg/udp/replay"
	"github.com/davecgh/go-spew/spew"
	"github.com/etcd-io/bbolt"
	"github.com/sirupsen/logrus"
	"time"

	"github.com/cj123/assetto-server-manager/pkg/udp"
)

var sessionFile = time.Now().Format("2006-01-02_15.04.db")

func main() {
	db, err := bbolt.Open(sessionFile, 0644, nil)

	if err != nil {
		logrus.WithError(err).Fatal("can't open bolt store")
	}

	callback := replay.RecordUDPMessages(db)

	udpServerConn, err := udp.NewServerClient("127.0.0.1", 12002, 11002, false, "", 0, func(response udp.Message) {
		logrus.Printf("%d %T", response.Event(), response)
		callback(response)
	})

	if err != nil {
		logrus.WithError(err).Fatal("can't record")
	}

	defer udpServerConn.Close()

	ch := make(chan struct{})

	<-ch // wait forever
}

func exampleReplay() {
	db, err := bbolt.Open("2019-04-05_11.41.db", 0644, nil)

	if err != nil {
		logrus.WithError(err).Fatal("can't open bolt store")
	}

	err = replay.ReplayUDPMessages(db, 1, func(response udp.Message) {
		spew.Dump(response)
	}, time.Second)

	if err != nil {
		panic(err)
	}
}
