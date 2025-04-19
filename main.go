package main

import (
	"fmt"

	"github.com/seyhpirr/go_bittorrent/peers"
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
	connectedPeers, err := peers.HandlePeersConcurrently(peerList, infoHash, peerID)
	if err != nil {
		fmt.Println("Error connecting to peers:", err)
	}

	fmt.Printf("Connected to %d peer(s):\n", len(connectedPeers))
	for _, p := range connectedPeers {
		fmt.Printf("Peer IP: %s, Port: %d, Peer ID: %x\n", p.IP, p.Port, p.PeerID)
	}

	//Download

}
