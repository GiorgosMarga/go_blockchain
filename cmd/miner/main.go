package main

import (
	"fmt"

	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/miner"
)

func main() {
	publicKey := crypto.Random()

	fmt.Printf("Dummy public key %x\n", publicKey)

	m := miner.New(":3000", publicKey)
	m.Start()
}
