package messages

import (
	"github.com/GiorgosMarga/blockchain/block"
	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/transaction"
	"github.com/GiorgosMarga/blockchain/utils"
)

type Message struct {
	From    string
	To      string
	Payload []byte
}

// messages types
type FetchUTXOs struct {
	PublicKey crypto.Hash
}

type UTXOs struct {
	Utxos map[crypto.Hash]utils.UtxoEntry
}

type SubmitTx struct {
	Tx transaction.Transaction
}

type NewTx struct {
	Tx transaction.Transaction
}

type FetchTemplate struct {
	PublicKey crypto.Hash
}
type Template struct {
	Block *block.Block
}

type ValidateTemplate struct {
	Block *block.Block
}

type TemplateValidity struct {
	IsValid bool
}

type SubmitTemplate struct {
	Block *block.Block
}
