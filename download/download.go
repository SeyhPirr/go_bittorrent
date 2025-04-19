package download

import . "github.com/seyhpirr/go_bittorrent/peers"

type TorrentMeta struct {
	PieceLength     int
	TotalLength     int
	NumPieces       int
	LastPieceLength int
}

var (
	torrentMeta       *TorrentMeta // Contains total pieces, piece length, etc.
	downloadedPieces  []bool       // Mark true if we already have it
	downloadingPieces = make(map[int]bool)
)

func DownloadTheFile(connectedPeers []*Peer) error {
	// do the appropriate messaging and download the file
}

func selectPieceToRequest(peer *Peer) (index, begin, length int, ok bool) {
	peer.Mutex.Lock()

	defer peer.Mutex.Unlock()

	for i := 0; i < len(peer.Bitfield)*8; i++ {
		if hasPiece(peer.Bitfield, i) && !downloadedPieces[i] && !downloadingPieces[i] {
			downloadingPieces[i] = true
			pieceLength := torrentMeta.PieceLength
			if i == torrentMeta.NumPieces-1 {
				pieceLength = torrentMeta.LastPieceLength
			}
			return i, 0, pieceLength, true
		}
	}
	return 0, 0, 0, false
}

func hasPiece(bitfield []byte, index int) bool {
	byteIndex := index / 8
	bitOffset := index % 8

	if byteIndex >= len(bitfield) {
		return false
	}
	bitPosition := 7 - bitOffset
	mask := byte(1 << bitPosition)
	targetByte := bitfield[byteIndex]
	return (targetByte & mask) != 0
}
