package main // or your node package

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/GiorgosMarga/blockchain/block"
	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/transaction"
)

// Helper functions for templates
var templateFuncs = template.FuncMap{
	"shortHash":   shortHash,
	"shortPubkey": shortPubkey,
	"timeFormat":  timeFormat,
	"containsTx":  containsTx,
	"txHash":      txHash,
	"outputHash":  outputHash,
}

// Helper function definitions
func shortHash(h crypto.Hash) string {
	if len(h) == 0 {
		return "0"
	}
	s := fmt.Sprintf("%x", h)
	// if len(s) > 16 {
	// 	return s[:8] + "..." + s[len(s)-8:]
	// }
	return s
}

func shortPubkey(pk []byte) string {
	if len(pk) == 0 {
		return "—"
	}
	s := fmt.Sprintf("%x", pk)
	return s
}

func outputHash(out *transaction.TxOutput) crypto.Hash { // make sure the package name matches your TxOutput
	return out.Hash()
}
func txHash(tx *transaction.Transaction) crypto.Hash {
	return tx.Hash() // your existing tx.Hash() method
}
func timeFormat(ts int64) string {
	if ts == 0 {
		return "—"
	}
	return time.UnixMilli(ts).Format("2006-01-02 15:04")
}

func containsTx(b *block.Block, query string) bool {
	if query == "" {
		return true
	}
	q := strings.ToLower(query)
	txIDStr := fmt.Sprintf("%x", b.Txs[0].Id) // you can loop all txs if needed
	return strings.Contains(strings.ToLower(txIDStr), q)
}
