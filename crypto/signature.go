package crypto

import (
	"crypto"
	"crypto/ecdsa"
)

type Signature []byte

func (s Signature) Verify(hash Hash, pubKey crypto.PublicKey) bool {
	k, ok := pubKey.(*ecdsa.PublicKey)
	if !ok {
		return false
	}
	return ecdsa.VerifyASN1(k, hash[:], s)
}
