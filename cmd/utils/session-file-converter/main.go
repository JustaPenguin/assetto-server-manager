package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/JustaPenguin/assetto-server-manager/pkg/udp/replay"
	"github.com/etcd-io/bbolt"
	"github.com/google/uuid"
)

var filename string

func init() {
	flag.StringVar(&filename, "f", "", "filename")
	flag.Parse()
}

func main() {
	f, err := os.Open(filename)

	if err != nil {
		panic(err)
	}

	defer f.Close()

	var entries []*replay.Entry

	if err := json.NewDecoder(f).Decode(&entries); err != nil {
		panic(err)
	}

	db, err := bbolt.Open(strings.TrimSuffix(filename, ".json")+".db", 0644, nil)

	if err != nil {
		panic(err)
	}

	defer db.Close()

	err = db.Update(func(tx *bbolt.Tx) error {
		bkt, err := tx.CreateBucketIfNotExists(replay.BucketName)

		if err != nil {
			return err
		}

		for _, entry := range entries {
			data, err := json.Marshal(entry)

			if err != nil {
				return err
			}

			if err := bkt.Put([]byte(uuid.New().String()), data); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		panic(err)
	}

	err = db.Sync()

	if err != nil {
		panic(err)
	}

	fmt.Printf("Successfully converted %d entries\n", len(entries))
}
