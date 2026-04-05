package blockchain

import (
	"testing"
	"time"

	"github.com/GiorgosMarga/blockchain/block"
	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/params"
	"github.com/GiorgosMarga/blockchain/transaction"
)

func TestLoadBlockchain(t *testing.T) {
	bobKp, err := crypto.LoadFromFile("../bob.priv.pem")
	if err != nil {
		t.Fatal(err)
	}
	// genesis tx
	pubKeyBytes  := bobKp.PublicKeyBytes()
	genesisTx := &transaction.Transaction{
		Id:  crypto.Zero(),
		Vin: []*transaction.TxInput{},
		Vout: []*transaction.TxOutput{
			{
				Value:     100_000_000,
				Id:        crypto.Zero(),
				PublicKey: pubKeyBytes,
			},
		},
	}
	merkleRoot := crypto.CalculateMerkleRoot([]*transaction.Transaction{genesisTx})
	// create genesis block
	block := block.Block{
		Header: block.Header{
			Timestamp:     time.Now().UnixMilli(),
			PrevBlockHash: crypto.Zero(),
			Nonce:         0,
			Target:        params.MyConfig.MinTarget,
			MerkleRoot:    merkleRoot,
		},
		Txs: []*transaction.Transaction{genesisTx},
	}
	block.BlockHash = block.Hash()

	b := New(params.MyConfig)
	if err := b.AddBlock(&block); err != nil {
		t.Fatal(err)
	}

	if err := b.LoadToFile("../genesis_blockchain"); err != nil {
		t.Fatal(err)
	}
}
