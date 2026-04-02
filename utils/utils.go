package utils

import "github.com/GiorgosMarga/blockchain/transaction"

type UtxoEntry struct {
	IsSpent  bool
	TxOutput *transaction.TxOutput
}
type MempoolEntry struct {
	TimestampMs int64
	Tx          *transaction.Transaction
}
