package peers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Peer struct {
	IP          net.IP
	Port        int
	Conn        net.Conn
	Choked      bool
	Bitfield    []byte
	PeerID      [20]byte
	MessageChan chan []byte
	mutex       sync.Mutex
	connected   bool
}

var (
	currentPeers []*Peer
	peersMu      sync.Mutex
)

type Handshake struct {
	Protocol string
	InfoHash [20]byte
	PeerID   [20]byte
	Reserved [8]byte
}

const (
	MsgChoke         = 0
	MsgUnchoke       = 1
	MsgInterested    = 2
	MsgNotInterested = 3
)

func connectToPeer(peerAddr string) (*Peer, error) {
	conn, err := net.DialTimeout("tcp", peerAddr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect peer %s: %v", peerAddr, err)
	}
	ipStr, portStr, err := net.SplitHostPort(peerAddr)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("invalid peer address %s: %v", peerAddr, err)
	}

	ip := net.ParseIP(ipStr)
	if ip == nil {
		conn.Close()
		return nil, fmt.Errorf("invalid IP address %s", ipStr)
	}

	port := 0
	_, err = fmt.Sscanf(portStr, "%d", &port)
	if err != nil || port < 1 || port > 65535 {
		conn.Close()
		return nil, fmt.Errorf("invalid port %s: %v", portStr, err)
	}

	return &Peer{
		IP:          ip,
		Port:        port,
		Conn:        conn,
		Choked:      true,                  // Default to choked state
		Bitfield:    nil,                   // Will be populated after handshake
		MessageChan: make(chan []byte, 10), // Buffered message channel
		connected:   true,                  // Mark as connected
		// PeerID will be set after successful handshake
	}, nil

}

func performHandshake(conn net.Conn, infoHash [20]byte, peerID [20]byte) (*Handshake, error) {
	handshake := bytes.NewBuffer(make([]byte, 0, 68))
	handshake.WriteByte(0x13)
	handshake.WriteString("BitTorrent protocol")
	handshake.Write(make([]byte, 8))
	handshake.Write(infoHash[:])
	handshake.Write(peerID[:])
	fmt.Printf("Raw handshake: %x\n", handshake)

	_, err := conn.Write(handshake.Bytes())

	if err != nil {
		return nil, fmt.Errorf("failed to send handshake: %v", err)

	}

	response := make([]byte, 68)
	_, err = io.ReadFull(conn, response)
	if err != nil {
		return nil, fmt.Errorf("failed to read handshake response: %v", err)
	}
	fmt.Printf("Received response from %s: %x\n", conn.RemoteAddr(), response)
	var peerHandshake Handshake
	reader := bytes.NewReader(response)
	protocolLength, _ := reader.ReadByte()
	protocol := make([]byte, protocolLength)
	reader.Read(protocol)
	reader.Read(peerHandshake.Reserved[:])
	reader.Read(peerHandshake.InfoHash[:])
	reader.Read(peerHandshake.PeerID[:])

	if !bytes.Equal(peerHandshake.InfoHash[:], infoHash[:]) {
		return nil, fmt.Errorf("mismatched info_hash")
	}

	return &peerHandshake, nil

}

func handleOnePeer(peerAddr string, infoHash [20]byte, peerID [20]byte) error {

	fmt.Println("Connecting to peer:", peerAddr)

	// Connect to the peer
	peer, err := connectToPeer(peerAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to peer %s: %v", peerAddr, err)
	}

	// Perform the handshake
	handshake, err := performHandshake(peer.Conn, infoHash, peerID)
	if err != nil {
		return fmt.Errorf("handshake failed with peer %s: %v", peerAddr, err)
	}
	// now peer struct is complete
	peer.PeerID = handshake.PeerID
	peer.connected = true

	//add the peer to the currentPeers if the handshake is succesfull
	peersMu.Lock()
	currentPeers = append(currentPeers, peer)
	peersMu.Unlock()
	fmt.Println("Handshake successful! Peer ID:", string(peer.PeerID[:]))

	// start listening for the peer

	for {
		msg, err := readMessage(peer.Conn)

		if err != nil {
			fmt.Printf("Error reading from %s: %v\n", peer.Conn.RemoteAddr(), err)
			break
		}
		if msg == nil {
			fmt.Printf("keep alive message")
			continue
		}
		peer.MessageChan <- msg.Payload
		handleMessage(peer, msg)
	}

	return nil
}

func HandlePeersConcurrently(peerList []string, infoHash [20]byte, peerID [20]byte) error {
	if len(peerList) == 0 {
		return fmt.Errorf("no peers available")
	}

	var (
		wg          sync.WaitGroup
		errChan     = make(chan error, len(peerList))
		peerChan    = make(chan string)
		ctx, cancel = context.WithCancel(context.Background())
	)
	defer cancel() // Ensure all resources are cleaned up

	// Worker goroutines with panic recovery
	worker := func() {
		defer wg.Done()
		for peerAddr := range peerChan {
			// Handle each peer with timeout
			err := handlePeerWithTimeout(ctx, peerAddr, infoHash, peerID)
			if err != nil {
				select {
				case errChan <- fmt.Errorf("peer %s: %w", peerAddr, err):
				case <-ctx.Done():
					return
				}
			} else {
				fmt.Printf("Successfully connected to peer %s\n", peerAddr)
			}
		}
	}

	// Start worker pool
	workerLimit := min(50, len(peerList)) // Don't create unnecessary workers
	for i := 0; i < workerLimit; i++ {
		wg.Add(1)
		go worker()
	}

	// Feed peers to workers
	go func() {
		defer close(peerChan)
		for _, peerAddr := range peerList {
			select {
			case peerChan <- peerAddr:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for completion
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Collect errors with context awareness
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
		if len(errors) > 10 { // Early exit if too many errors
			cancel()
			break
		}
	}

	// Analyze results
	switch {
	case len(errors) == len(peerList):
		return fmt.Errorf("all %d peers failed: %w", len(peerList), errors[0])
	case len(errors) > 0:
		return fmt.Errorf("%d/%d peers failed. First error: %w",
			len(errors), len(peerList), errors[0])
	default:
		return nil
	}
}

func handlePeerWithTimeout(ctx context.Context, addr string, infoHash [20]byte, peerID [20]byte) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() { done <- handleOnePeer(addr, infoHash, peerID) }()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("timeout: %w", ctx.Err())
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
