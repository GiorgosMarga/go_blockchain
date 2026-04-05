package block

import (
	"crypto/sha256"
	"encoding/binary"
	"time"

	"github.com/GiorgosMarga/blockchain/crypto"
)

type Header struct {
	Timestamp     int64
	PrevBlockHash crypto.Hash
	MerkleRoot    crypto.Hash
	Nonce         uint64
	Target        crypto.Hash
	Difficulty    int
}

func (h *Header) Bytes() []byte {
	buf := make([]byte, 0, 16+3*32)
	buf = binary.LittleEndian.AppendUint64(buf, uint64(h.Timestamp))
	buf = binary.LittleEndian.AppendUint64(buf, h.Nonce)
	buf = append(buf, h.PrevBlockHash[:]...)
	buf = append(buf, h.MerkleRoot[:]...)
	buf = append(buf, h.Target[:]...)
	return buf
}

func (h *Header) CalculateHash() crypto.Hash {
	return sha256.Sum256(h.Bytes())
}

func (h *Header) Mine(steps int) bool {
	h.Nonce = 0
	h.Timestamp = time.Now().UnixMilli()
	for range steps {
		hash := h.CalculateHash()
		if hash.Matches(h.Target) {
			return true
		}
		h.Nonce++
		if h.Nonce == 0 {
			h.Timestamp = time.Now().UnixMilli()
		}
	}
	return false
}
