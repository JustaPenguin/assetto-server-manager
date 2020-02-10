// This code would not have been possible without the two following projects:
//
// * AcTools (and Content Manager)
//   - https://github.com/gro-ove/actools - Licensed under the Microsoft Public License (Ms-PL)
//
// * Luigi Auriemma's QuickBMS, specifically the "Assetto Corsa ACD" script.
//   - https://aluigi.altervista.org/bms/assetto_corsa_acd.bms
//   - https://aluigi.altervista.org/quickbms.htm
package acd

import (
	"encoding/binary"
	"fmt"
	"io"

	"golang.org/x/text/encoding/unicode/utf32"
)

// Reader is a reader for Assetto Corsa .acd files
type Reader struct {
	r                io.ReadSeeker
	parentFolderName string

	Files []*File
}

// File is contained within an Assetto Corsa .acd archive.
type File struct {
	length      int32
	start       int64
	reader      io.ReadSeeker
	decipherKey string

	name string
}

// Bytes returns the data of the file
func (f *File) Bytes() ([]byte, error) {
	out := make([]byte, f.length)

	if _, err := f.reader.Seek(f.start, io.SeekStart); err != nil {
		return nil, err
	}

	if err := binary.Read(f.reader, binary.LittleEndian, &out); err != nil {
		return nil, err
	}

	decipher(out, f.decipherKey)

	utf32Decoded, err := utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM).NewDecoder().Bytes(out)

	if err != nil {
		return nil, err
	}

	return utf32Decoded, nil
}

// Name is the filename per the archive
func (f *File) Name() string {
	return f.name
}

// NewReader creates a reader for a given io.Reader. parentFolderName must be the original parent folder name
// as it is used for deciphering purposes
func NewReader(r io.ReadSeeker, parentFolderName string) (*Reader, error) {
	x := &Reader{
		r:                r,
		parentFolderName: parentFolderName,
	}

	err := x.init()

	if err != nil {
		return nil, err
	}

	return x, nil
}

func (r *Reader) init() error {
	key := cipherKey(r.parentFolderName)

	for {
		f, err := r.nextFileInfo()

		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		f.decipherKey = key

		r.Files = append(r.Files, f)
	}

	return nil
}

func (r *Reader) nextFileInfo() (*File, error) {
	var strlen int32

	err := binary.Read(r.r, binary.LittleEndian, &strlen)

	if err != nil {
		return nil, err
	}

	// -1111 appears to be a magic number of sorts.
	if strlen == -1111 {
		// it signifies that the next int32 should be thrown away, _then_ we get the strlen
		var throwaway int32

		err := binary.Read(r.r, binary.LittleEndian, &throwaway)

		if err != nil {
			return nil, err
		}

		err = binary.Read(r.r, binary.LittleEndian, &strlen)

		if err != nil {
			return nil, err
		}
	}

	name := make([]byte, strlen)

	if err := binary.Read(r.r, binary.LittleEndian, &name); err != nil {
		return nil, err
	}

	var length int32

	if err := binary.Read(r.r, binary.LittleEndian, &length); err != nil {
		return nil, err
	}

	pos, err := r.r.Seek(0, io.SeekCurrent)

	if err != nil {
		return nil, err
	}

	_, err = r.r.Seek(int64(length*4), io.SeekCurrent)

	if err != nil {
		return nil, err
	}

	return &File{
		length: length * 4,
		start:  pos,
		name:   string(name),
		reader: r.r,
	}, nil
}

// cipherKey generates an encryption key from the folder of the given filename
func cipherKey(filename string) string {
	part1 := 0

	for i := 0; i < len(filename); i++ {
		part1 += int(filename[i])
	}

	part2 := 0

	for i := 0; i < len(filename)-1; i++ {
		part2 *= int(filename[i])

		i++

		part2 -= int(filename[i])
	}

	part3 := 0

	for i := 1; i < len(filename)-3; i += 4 {
		part3 *= int(filename[i])

		i++

		part3 /= int(filename[i] + 0x1b)

		i -= 2

		part3 += -0x1b - int(filename[i])
	}

	part4 := 0x1683

	for i := 1; i < len(filename); i++ {
		part4 -= int(filename[i])
	}

	part5 := 0x42

	for i := 1; i < len(filename)-4; i += 4 {
		n := int(filename[i]+0xf) * part5
		i--
		x := int(filename[i])

		i++

		x += 0xf
		x *= n
		x += 0x16
		part5 = x
	}

	part6 := 0x65

	for i := 0; i < len(filename)-2; i += 2 {
		part6 -= int(filename[i])
	}

	part7 := 0xab

	for i := 0; i < len(filename)-2; i += 2 {
		part7 %= int(filename[i])
	}

	part8 := 0xab

	for i := 0; i < len(filename)-1; i++ {
		tmp := int(filename[i])
		part8 /= tmp
		i++
		tmp2 := int(filename[i])
		part8 += tmp2
		i--
	}

	part1 &= 0xff
	part2 &= 0xff
	part3 &= 0xff
	part4 &= 0xff
	part5 &= 0xff
	part6 &= 0xff
	part7 &= 0xff
	part8 &= 0xff

	return fmt.Sprintf("%d-%d-%d-%d-%d-%d-%d-%d", part1, part2, part3, part4, part5, part6, part7, part8)
}

// decipher a []byte given its key
func decipher(out []byte, key string) {
	x := 0

	for b := 0; b < len(out); b++ {
		if out[b] == 0 {
			continue
		}

		out[b] -= key[x%len(key)]
		x++
	}
}
