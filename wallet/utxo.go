package wallet

import (
	"crypto/ecdsa"

	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/utils"
)

type UtxoStore struct {
	MyKeys []crypto.KeyPair
	Utxos  map[ecdsa.PublicKey][]utils.UtxoEntry
}
