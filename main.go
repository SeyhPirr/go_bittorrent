package main

import (
	"fmt"
	"strings"

	peers "github.com/seyhpirr/go_bittorrent/peers"
	tracker "github.com/seyhpirr/go_bittorrent/tracker"
)

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
		// Enhanced error handling
		fmt.Printf("\n⚠️ Peer Connection Report ⚠️\n")
		fmt.Printf("Total peers attempted: %d\n", len(peerList))

		// Unwrap the error to get detailed information
		if multiErr, ok := err.(interface{ Unwrap() []error }); ok {
			// This is a multi-error case
			fmt.Printf("Failed connections: %d\n", len(multiErr.Unwrap()))
			fmt.Println("First few errors:")
			for i, e := range multiErr.Unwrap() {
				if i >= 3 { // Show only first 3 errors to avoid clutter
					fmt.Printf("... and %d more errors\n", len(multiErr.Unwrap())-i)
					break
				}
				fmt.Printf("- %v\n", e)
			}
		} else {
			// Single error case
			fmt.Printf("Error: %v\n", err)
		}

		// If no peers connected at all
		if len(peerList) > 0 && strings.Contains(err.Error(), "all peers failed") {
			fmt.Println("\n❌ Critical: All peer connections failed!")
			fmt.Println("This usually indicates:")
			fmt.Println("- Your client is blocked (check firewall)")
			fmt.Println("- The torrent is no longer seeded")
			fmt.Println("- Network configuration issues")
		}
	}
	select {}
}
