package main

import (
	"fmt"
	"strings"

	peers "github.com/seyhpirr/go_bittorrent/peers"
	tracker "github.com/seyhpirr/go_bittorrent/tracker"
)

// Handle peer connection errors with more clarity and simplicity
func handlePeerConnectionError(err error, peerList []string) {
	fmt.Printf("\n⚠️ Peer Connection Report ⚠️\n")
	fmt.Printf("Total peers attempted: %d\n", len(peerList))

	if err == nil {
		return
	}

	// Check if it's a multi-error case
	if multiErr, ok := err.(interface{ Unwrap() []error }); ok {
		handleMultiError(multiErr)
	} else {
		// Handle single error
		fmt.Printf("Error: %v\n", err)
	}

	// Check if all peers failed
	if len(peerList) > 0 && strings.Contains(err.Error(), "all peers failed") {
		handleCriticalError()
	}
}

// Handle multi-error case
func handleMultiError(multiErr interface{ Unwrap() []error }) {
	failedConnections := multiErr.Unwrap()
	fmt.Printf("Failed connections: %d\n", len(failedConnections))
	fmt.Println("First few errors:")

	for i, e := range failedConnections {
		if i >= 3 { // Show only first 3 errors
			fmt.Printf("... and %d more errors\n", len(failedConnections)-i)
			break
		}
		fmt.Printf("- %v\n", e)
	}
}

// Handle the case when all peer connections failed
func handleCriticalError() {
	fmt.Println("\n❌ Critical: All peer connections failed!")
	fmt.Println("This usually indicates:")
	fmt.Println("- Your client is blocked (check firewall)")
	fmt.Println("- The torrent is no longer seeded")
	fmt.Println("- Network configuration issues")
}

func main() {
	//handle bencoding part
	theTorrentFile, err := tracker.HandleBencoding("debian-iso.torrent")
	if err != nil {
		fmt.Println("Failed to process torrent file:", err)
		return
	}
	fmt.Println("Announce URL:", theTorrentFile.Announce)

	//Create A Peer ID
	peerID := tracker.GeneratePeerID()
	//handle tracker part
	peerList, err := tracker.HandleTracker(*theTorrentFile, peerID)
	if err != nil {
		fmt.Println("Tracker error:", err)
		return
	}
	fmt.Println("Peers received:", peerList)

	infoHash, err := tracker.ComputeInfoHash(theTorrentFile.Info)
	if err != nil {
		println("error with computing info hash:", err)
	}
	fmt.Println("infohash:", infoHash)

	err = peers.HandlePeersConcurrently(peerList, infoHash, peerID)
	if err != nil {
		handlePeerConnectionError(err, peerList)
	}
	fmt.Println("get the peer list is being executed")
	peers.GetThePeerList()
	peers.WaitForPeerHandlers()
}
