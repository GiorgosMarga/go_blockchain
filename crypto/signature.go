package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
)

type Signature []byte

func (s Signature) Verify(hash Hash, pubKey []byte) bool {
	x, y := elliptic.UnmarshalCompressed(elliptic.P256(), pubKey)
	k := &ecdsa.PublicKey{
		Curve: elliptic.P256(),
		X:     x,
		Y:     y,
	}
	return ecdsa.VerifyASN1(k, hash[:], s)
}
