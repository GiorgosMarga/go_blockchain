package block

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
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
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(h); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func (h *Header) CalculateHash() crypto.Hash {
	return sha256.Sum256(h.Bytes())
}

func (h *Header) Mine(steps int) bool {
	h.Nonce = 0
	h.Timestamp = time.Now().Unix()
	for range steps {
		hash := h.CalculateHash()
		if hash.Matches(h.Target) {
			return true
		}
		h.Nonce++
		if h.Nonce == 0 {
			h.Timestamp = time.Now().Unix()
		}
	}
	return false
}
