package wallet

import (
	"crypto/ecdsa"

	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/utils"
)

type UtxoStore struct {
	myKeys []crypto.KeyPair
	utxos  map[ecdsa.PublicKey][]utils.UtxoEntry
}
