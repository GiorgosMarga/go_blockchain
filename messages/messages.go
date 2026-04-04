package messages

import (
	"crypto/ecdsa"

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
type FetchUTXOsReq struct {
	FromAddr  string
	PublicKey ecdsa.PublicKey
}

type UTXOsResp struct {
	FromAddr string
	Utxos    []utils.UtxoEntry
}

type NewTx struct {
	Tx *transaction.Transaction
}

type FetchTemplate struct {
	PublicKey crypto.Hash
	FromAddr string
}
type Template struct {
	Block *block.Block
}

type ValidateTemplateReq struct {
	Block    *block.Block
	FromAddr string
}

type ValidateTemplateResp struct {
	IsValid  bool
	FromAddr string
}

type SubmitTemplate struct {
	Block *block.Block
}
type SubmitTransaction struct {
	Tx *transaction.Transaction
}
type FetchBlockReq struct {
	Height   int
	FromAddr string
}
type FetchBlockResp struct {
	Block    *block.Block
	Height   int
	FromAddr string
}

type DifferenceReq struct {
	Height   int
	FromAddr string
}

type DifferenceResp struct {
	Height   int
	FromAddr string
}

type NewBlock struct {
	Block *block.Block
}
