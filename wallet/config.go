package wallet

import (
	"fmt"

	"github.com/GiorgosMarga/blockchain/crypto"
)

type FeeType byte

const (
	Fixed FeeType = iota
	Percent
)

type FeeConfig struct {
	FeeType FeeType
	Value   float64
}

type Config struct {
	MyKeys      []crypto.KeyPair
	Contacts    []Recipient
	DefaultNode string
	FeeConfig   FeeConfig
}

func DummyConfig() Config {
	myKp, err := crypto.LoadFromFile("bob.priv.pem")
	if err != nil {
		panic(err)
	}
	fmt.Printf("[Wallet]: Public Key %x\n", myKp.PublicKeyBytes())

	r1, _ := NewRecipient("Alice", "alice.pub.pem")
	return Config{
		MyKeys: []crypto.KeyPair{myKp},
		Contacts: []Recipient{
			r1,
		},
		DefaultNode: ":3000",
		FeeConfig: FeeConfig{
			FeeType: Percent,
			Value:   0.1,
		},
	}
}
