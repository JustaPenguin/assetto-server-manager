package csp

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/JustaPenguin/assetto-server-manager/pkg/udp"
)

type Command interface {
	GetMessageType() uint16
}

func ToChatMessage(carID udp.CarID, c Command) (udp.Message, error) {
	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.LittleEndian, c.GetMessageType())

	if err != nil {
		return nil, err
	}

	err = binary.Write(buf, binary.LittleEndian, c)

	if err != nil {
		return nil, err
	}

	fmt.Printf("%x\n\n", buf.Bytes())

	enc := base64.StdEncoding.EncodeToString(buf.Bytes())

	message := fmt.Sprintf("\t\t\t\t$CSP0:%s", enc)

	for strings.HasSuffix(message, "=") {
		message = strings.TrimSuffix(message, "=")
	}

	fmt.Println("sending", message)

	return udp.NewSendChat(carID, message)
}

func FromChatMessage(message string) (Command, error) {
	fmt.Println("GOT CHAT MESSAGE", message) // @TODO

	return nil, nil
}

type HandshakeSend struct {
	MinVersion        uint32
	RequiresWeatherFX bool
}

func (h HandshakeSend) GetMessageType() uint16 {
	return 0
}

type HandshakeResponse struct {
	Version           uint32
	IsWeatherFXActive bool
}

func (h HandshakeResponse) GetMessageType() uint16 {
	return 1
}
