package acd

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Reader struct {
	r io.Reader

	File []*File
}

type File struct {
	data []byte
	name string
}

func (f *File) Open() (io.ReadCloser, error) {
	return nil, nil // @TODO
}

func NewReader(r io.Reader) (*Reader, error) {
	x := &Reader{
		r: r,
	}

	err := x.init()

	if err != nil {
		return nil, err
	}

	return x, err
}

func (r *Reader) init() error {

	for {
		var strlen uint32

		if err := binary.Read(r.r, binary.LittleEndian, &strlen); err != nil {
			return err
		}

		fName := make([]byte, strlen)

		if err := binary.Read(r.r, binary.LittleEndian, &fName); err != nil {
			return err
		}

		var length uint32

		if err := binary.Read(r.r, binary.LittleEndian, &length); err != nil {
			return err
		}

		out := make([]byte, length*4)

		if err := binary.Read(r.r, binary.LittleEndian, &out); err != nil {
			return err
		}

		if string(fName) != "tyres.ini" {
			continue
		}

		// "assettocorsa\\content\\cars\\rss_formula_
		decrypt(out, "247-76-61-254-174-237-57-53")


		fmt.Println(string(out))
	}

	return nil
}

func encryptionKey(filename string) string {

}

func decrypt(out []byte, key string) {
	x := 0

	for b := 0; b < len(out); b++ {
		if out[b] == 0 {
			continue
		}

		out[b] = out[b] - key[x % len(key)]

		x++
	}
}