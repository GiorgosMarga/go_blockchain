package crypto

import (
	"crypto/rand"
	"math/big"
)

type Hash [32]byte

func (h Hash) Matches(target Hash) bool {
	hInt := new(big.Int)
	hInt.SetBytes(h[:])
	targetInt := new(big.Int)
	targetInt.SetBytes(target[:])
	return hInt.Cmp(targetInt) == -1
}

func Zero() Hash {
	return [32]byte{}
}

func Random() Hash {
	randomBytes := [32]byte{}
	rand.Read(randomBytes[:])
	return randomBytes
}
