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
	if msg == nil {
		fmt.Printf("⚠️ Got nil message from peer %s, skipping.\n", peer.IP.String())
		return
	}

	switch msg.ID {
	case 0: // choke
		fmt.Println("⛔ Received choke from", peer.IP)
		peer.mutex.Lock()
		peer.Choked = true
		peer.mutex.Unlock()

	case 1: // unchoke
		fmt.Println("✅ Received unchoke from", peer.IP)
		peer.mutex.Lock()
		peer.Choked = false
		peer.mutex.Unlock()
		// send a request for a piece here if needed

	case 4: // have
		if len(msg.Payload) < 4 {
			fmt.Printf("⚠️ Invalid 'have' message from %s (too short)\n", peer.IP)
			return
		}
		pieceIndex := binary.BigEndian.Uint32(msg.Payload)
		fmt.Printf("📢 Peer %s has piece %d\n", peer.IP, pieceIndex)
		// Optionally mark the piece as available in some global state or request it

	case 5: // bitfield
		fmt.Println("🧩 Received bitfield from", peer.IP)
		peer.mutex.Lock()
		peer.Bitfield = msg.Payload
		peer.mutex.Unlock()
		// Evaluate which pieces to request based on your strategy

	case 7: // piece
		fmt.Printf("📦 Received piece from %s\n", peer.IP)
		// parse and store piece (msg.Payload will contain: index, begin, block)

	default:
		fmt.Printf("🤷 Unknown message ID %d from %s\n", msg.ID, peer.IP)
	}
}

func GetThePeerList() {
	// a function that prints the peer list

	peersMu.Lock()
	defer peersMu.Unlock()

	fmt.Println("🌐 Connected Peers List:")
	if len(currentPeers) == 0 {
		fmt.Println("No connected peers.")
		return
	}

	for i, peer := range currentPeers {
		status := "Connected"
		if !peer.connected {
			status = "Disconnected"
		}
		fmt.Printf("%d. IP: %s | Port: %d | Choked: %t | Status: %s\n",
			i+1, peer.IP.String(), peer.Port, peer.Choked, status)
	}
}
