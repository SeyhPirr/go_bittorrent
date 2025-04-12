package torrent

import (
	"fmt"
	"os"

	"github.com/jackpal/bencode-go"
)

type TorrentFile struct {
	Announce     string      `bencode:"announce"`
	Info         TorrentInfo `bencode:"info"`
	CreationDate int64       `bencode:"creation date"`
	Comment      string      `bencode:"comment"`
	CreatedBy    string      `bencode:"created by"`
	Encoding     string      `bencode:"encoding"`
}

type TorrentInfo struct {
	Files       []File `bencode:"files,omitempty"`  // For multi-file torrents
	Length      int    `bencode:"length,omitempty"` // For single-file torrents
	Name        string `bencode:"name"`
	PieceLength int    `bencode:"piece length"`
	Pieces      string `bencode:"pieces"`
	Private     int    `bencode:"private,omitempty"`
}

type File struct {
	Length int      `bencode:"length"`
	Path   []string `bencode:"path"`
}

func HandleBencoding(filePath string) (*TorrentFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening torrent file: %w", err)
	}
	defer file.Close()

	fmt.Println("Successfully opened the BitTorrent file.")

	var torrent TorrentFile
	err = bencode.Unmarshal(file, &torrent)
	if err != nil {
		return nil, fmt.Errorf("error decoding .torrent file: %w", err)
	}

	fmt.Println("Successfully decoded the BitTorrent file.")
	return &torrent, nil
}
