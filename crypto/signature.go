package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
)

type Signature []byte

func (s Signature) Verify(hash Hash, pubKey []byte) bool {
	k, _ := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), pubKey)
	return ecdsa.VerifyASN1(k, hash[:], s)
}
