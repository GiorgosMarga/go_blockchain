package transaction

import (
	"bytes"
	"encoding/gob"

	"github.com/GiorgosMarga/blockchain/crypto"
)

type TxInput struct {
	PrevTxOutputHash crypto.Hash
	Signature        crypto.Signature
}
type TxOutput struct {
	Value     uint64
	Id        string
	PublicKey crypto.Hash
}

func (tout *TxOutput) Hash() crypto.Hash {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(tout); err != nil {
		panic(err)
	}
	return crypto.Hash(buf.Bytes())
}

type Transaction struct {
	Id   crypto.Hash
	Vin  []*TxInput
	Vout []*TxOutput
}

func (tx *Transaction) Hash() crypto.Hash {
	return crypto.Zero()
}
