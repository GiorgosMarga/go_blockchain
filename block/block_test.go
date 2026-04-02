package block

import (
	"fmt"
	"testing"
	"time"

	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/params"
)

func TestMine(t *testing.T) {
	b := Block{
		Header: Header{
			Timestamp:     time.Now().Unix(),
			PrevBlockHash: crypto.Zero(),
			MerkleRoot:    crypto.Zero(),
			Nonce:         0,
			Difficulty:    4,
		},
	}

	for !b.Mine(1_000_000, params.MyConfig.MinTarget) {
	}
	fmt.Printf("%+v\n%x\n", b.Header, b.Header.CalculateHash())
}
