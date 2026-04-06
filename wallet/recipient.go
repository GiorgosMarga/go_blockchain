package wallet

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/GiorgosMarga/blockchain/crypto"
)

type Recipient struct {
	Name       string
	PubKeyPath string
	PubKey     ecdsa.PublicKey
	Address    string
}

func NewRecipient(name, keyPath string) (Recipient, error) {
	kp, err := crypto.LoadPubliFromFile(keyPath)
	if err != nil {
		fmt.Println(err)
		return Recipient{}, err
	}
	return Recipient{
		Name:       name,
		PubKeyPath: keyPath,
		PubKey:     kp.PublicKey,
		Address:    fmt.Sprintf("%x", kp.PublicKeyBytes()),
	}, nil
}
