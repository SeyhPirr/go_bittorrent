package download

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	. "github.com/seyhpirr/go_bittorrent/peers"
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
	if msg == nil {
		fmt.Printf("⚠️ Got nil message from peer %s, skipping.\n", peer.IP.String())
		return
	}

	switch msg.ID {
	case 0: // choke
		fmt.Println("⛔ Received choke from", peer.IP)
		peer.Mutex.Lock()
		peer.Choked = true
		peer.Mutex.Unlock()

	case 1: // unchoke
		fmt.Println("✅ Received unchoke from", peer.IP)
		peer.Mutex.Lock()
		peer.Choked = false
		peer.Mutex.Unlock()
		// send a request for a piece here if needed

	case 4: // have
		if len(msg.Payload) < 4 {
			fmt.Printf("⚠️ Invalid 'have' message from %s (too short)\n", peer.IP)
			return
		}
		pieceIndex := binary.BigEndian.Uint32(msg.Payload)
		fmt.Printf("📢 Peer %s has piece %d\n", peer.IP, pieceIndex)
		// send interested message

	case 5: // bitfield
		fmt.Println("🧩 Received bitfield from", peer.IP)
		peer.Mutex.Lock()
		peer.Bitfield = msg.Payload
		peer.Mutex.Unlock()

	case 7: // piece
		fmt.Printf("📦 Received piece from %s\n", peer.IP)
		// parse and store piece (msg.Payload will contain: index, begin, block)

	default:
		fmt.Printf("🤷 Unknown message ID %d from %s\n", msg.ID, peer.IP)
	}
}

func sendMessage(conn net.Conn, msg *Message) error {
	length := uint32(1 + len(msg.Payload))
	buf := make([]byte, 4+length)

	// write length prefix
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = msg.ID

	// write payload if present
	if len(msg.Payload) > 0 {
		copy(buf[5:], msg.Payload)
	}
	_, err := conn.Write(buf)

	return err
}

func sendInterested(conn net.Conn) error {
	msg := &Message{
		ID:      2,
		Payload: nil,
	}
	return sendMessage(conn, msg)
}

func sendNotInterested(conn net.Conn) error {
	msg := &Message{
		ID:      3,
		Payload: nil,
	}
	return sendMessage(conn, msg)
}

func sendRequest(conn net.Conn, index, begin, length int) error {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	msg := &Message{
		ID:      6, // request
		Payload: payload,
	}
	return sendMessage(conn, msg)
}
