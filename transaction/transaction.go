package transaction

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/GiorgosMarga/blockchain/crypto"
)

type TxInput struct {
	PrevTxOutputHash crypto.Hash
	Signature        crypto.Signature
}

func (tin *TxInput) bytes() []byte {
	buf := make([]byte, 32)
	copy(buf, tin.PrevTxOutputHash[:])
	buf = binary.LittleEndian.AppendUint32(buf[32:], uint32(len(tin.Signature)))
	buf = append(buf, tin.Signature...)
	return buf
}

type TxOutput struct {
	Value     uint64
	Id        crypto.Hash
	PublicKey []byte
}

func (tout *TxOutput) bytes() []byte {
	buf := make([]byte, 40+len(tout.PublicKey)) // 8 + 32 + 32
	binary.LittleEndian.PutUint64(buf, tout.Value)
	copy(buf[8:], tout.Id[:])
	copy(buf[40:], tout.PublicKey)
	return buf
}
func (tout *TxOutput) Hash() crypto.Hash {
	return crypto.Hash(sha256.Sum256(tout.bytes()))
}

type Transaction struct {
	Id   crypto.Hash
	Vin  []*TxInput
	Vout []*TxOutput
}

func (tx *Transaction) Hash() crypto.Hash {

	buf := make([]byte, 32)
	copy(buf, tx.Id[:])
	for _, tin := range tx.Vin {
		buf = append(buf, tin.bytes()...)
	}
	for _, tout := range tx.Vout {
		buf = append(buf, tout.bytes()...)
	}
	return crypto.Hash(sha256.Sum256(buf))
}
