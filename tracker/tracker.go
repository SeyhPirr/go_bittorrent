package torrent

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"

	"github.com/jackpal/bencode-go"
)

type TrackerResponse struct {
	Interval      int    `bencode:"interval"`
	Peers         string `bencode:"peers"`
	FailureReason string `bencode:"failure reason"`
}

func GeneratePeerID() [20]byte {
	const prefix = "-TR4050-"
	var peerID [20]byte

	// Copy prefix into the first 8 bytes
	copy(peerID[:], prefix)

	// Generate 12 random bytes and copy them into the rest of peerID
	_, err := rand.Read(peerID[8:])
	if err != nil {
		panic("failed to generate peer ID: " + err.Error()) // Panic or handle error
	}

	return peerID
}
func urlEncodePeerID(peerID [20]byte) string {

	encoded := base64.URLEncoding.EncodeToString(peerID[:])
	return encoded[:20] // Trim to 12 characters
}
func ComputeInfoHash(info TorrentInfo) ([20]byte, error) {
	var buf bytes.Buffer
	if err := bencode.Marshal(&buf, info); err != nil {
		return [20]byte{}, err
	}

	hash := sha1.Sum(buf.Bytes())
	fmt.Println("Computed info_hash (hex):", hex.EncodeToString(hash[:]))
	return hash, nil
}

func connectToTracker(announceURL string, infoHash [20]byte, totalLength int, peerID [20]byte) ([]string, error) {
	baseURL, err := url.Parse(announceURL)
	if err != nil {
		return nil, fmt.Errorf("invalid tracker URL: %v", err)
	}
	query := baseURL.Query()
	query.Set("info_hash", string(infoHash[:]))
	urlEncodedPeerID := urlEncodePeerID(peerID)
	query.Set("peer_id", urlEncodedPeerID)
	query.Set("port", "6882")
	query.Set("uploaded", "0")
	query.Set("downloaded", "0")
	query.Set("left", fmt.Sprint(totalLength))
	query.Set("compact", "1")
	baseURL.RawQuery = query.Encode()

	// Log the exact URL being requested
	fmt.Println("Tracker request URL:", baseURL.String())

	client := &http.Client{}
	req, err := http.NewRequest("GET", baseURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("User-Agent", "MyGoClient/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tracker responded with status: %s", resp.Status)
	}

	var trackerResp TrackerResponse
	err = bencode.Unmarshal(resp.Body, &trackerResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tracker response: %v", err)
	}

	if trackerResp.FailureReason != "" {
		return nil, fmt.Errorf("tracker error: %s", trackerResp.FailureReason)
	}

	peers := parseCompactPeerList(trackerResp.Peers)
	return peers, nil
}

func parseCompactPeerList(peerData string) []string {
	var peers []string
	for i := 0; i < len(peerData); i += 6 {
		ip := peerData[i : i+4]
		port := peerData[i+4 : i+6]
		peers = append(peers, fmt.Sprintf("%d.%d.%d.%d:%d", ip[0], ip[1], ip[2], ip[3], uint16(port[0])<<8|uint16(port[1])))
	}
	return peers
}

func HandleTracker(theTorrentFile TorrentFile, peerID [20]byte) ([]string, error) {
	info := theTorrentFile.Info

	hash, err := ComputeInfoHash(info)
	if err != nil {
		return nil, fmt.Errorf("failed to compute info hash: %w", err)
	}

	peerList, err := connectToTracker(theTorrentFile.Announce, hash, theTorrentFile.Info.Length, peerID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tracker: %w", err)
	}

	if len(peerList) == 0 {
		return nil, fmt.Errorf("no peers found from tracker")
	}

	return peerList, nil
}
