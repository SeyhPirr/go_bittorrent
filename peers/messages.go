package peers

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
)

type Message struct {
	ID      byte
	Payload []byte
}

func readMessage(conn net.Conn) (*Message, error) {
	//Read the length prefix(4bytes)
	lentgthBuf := make([]byte, 4)
	_, err := io.ReadFull(conn, lentgthBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read message length: %w", err)
	}
	length := binary.BigEndian.Uint32(lentgthBuf)

	// handle keep alive message
	if length == 0 {
		return nil, nil
	}

	//read meesage id
	idBuf := make([]byte, 1)
	_, err = io.ReadFull(conn, idBuf)
	if err != nil {
		return nil, fmt.Errorf("failed to read message ID: %w", err)
	}
	messageID := idBuf[0]

	// read the payload
	payloadLength := int(length) - 1
	payload := make([]byte, payloadLength)
	if payloadLength > 0 {
		_, err = io.ReadFull(conn, payload)
		if err != nil {
			return nil, fmt.Errorf("failed to read payload: %w", err)
		}
	}

	return &Message{
		ID:      messageID,
		Payload: payload,
	}, nil

}

func handleMessage(peer *Peer, msg *Message) {
	switch msg.ID {
	case 0:
		fmt.Println("Received choke from", peer.PeerID)
	case 1:
		fmt.Println("Received unchoke from", peer.PeerID)
	case 5:
		fmt.Println("Received bitfield")
		// parse and store the bitfield
	case 7:
		fmt.Println("Received piece")
		// handle piece data
	default:
		fmt.Printf("Received message ID %d from %s\n", msg.ID, peer.PeerID)
	}
}
