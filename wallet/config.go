package wallet

import "github.com/GiorgosMarga/blockchain/crypto"

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
	r1, _ := NewRecipient("Alice", "alice.pub.pem")
	r2, _ := NewRecipient("Bob", "bob.pub.pem")
	return Config{
		MyKeys: make([]crypto.KeyPair, 0),
		Contacts: []Recipient{
			r1, r2,
		},
		DefaultNode: ":3000",
		FeeConfig: FeeConfig{
			FeeType: Percent,
			Value:   0.1,
		},
	}
}
