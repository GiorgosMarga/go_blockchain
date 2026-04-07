package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

type KeyPair struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  ecdsa.PublicKey
}

func NewKeyPair() KeyPair {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	return KeyPair{
		PrivateKey: privateKey,
		PublicKey:  privateKey.PublicKey,
	}
}
func (kp *KeyPair) PublicKeyBytes() []byte {
	b, _ := kp.PublicKey.Bytes()
	return b
}

func (kp KeyPair) Sign(hash Hash) ([]byte, error) {
	return ecdsa.SignASN1(rand.Reader, kp.PrivateKey, hash[:])
}

func LoadFromFile(filename string) (KeyPair, error) {
	d, err := os.ReadFile(filename)
	if err != nil {
		return KeyPair{}, err
	}
	block, _ := pem.Decode(d)
	if block == nil {
		return KeyPair{}, fmt.Errorf("failed to decode PEM")
	}
	priv, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{
		PrivateKey: priv,
		PublicKey:  priv.PublicKey,
	}, nil
}
func LoadPubliFromFile(filename string) (KeyPair, error) {
	d, err := os.ReadFile(filename)
	if err != nil {
		return KeyPair{}, err
	}
	block, _ := pem.Decode(d)
	if block == nil {
		return KeyPair{}, fmt.Errorf("failed to decode PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return KeyPair{}, err
	}
	pubKey := pub.(*ecdsa.PublicKey)
	return KeyPair{
		PublicKey: *pubKey,
	}, nil
}
func (kp KeyPair) LoadToFile(filepath string) error {
	// convert the private key to DER (binary) format
	derBytes, err := x509.MarshalECPrivateKey(kp.PrivateKey)
	if err != nil {
		return err
	}

	// create a PEM block
	block := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: derBytes,
	}

	// create the file and write the PEM data
	privfile, err := os.Create(fmt.Sprintf("%s.priv.pem", filepath))
	if err != nil {
		return err
	}
	defer privfile.Close()

	if err := pem.Encode(privfile, block); err != nil {
		return err
	}

	derBytes, err = x509.MarshalPKIXPublicKey(&kp.PublicKey)
	if err != nil {
		return err
	}

	// create a PEM block
	block = &pem.Block{
		Type:  "EC PUBLIC KEY",
		Bytes: derBytes,
	}

	pubfile, err := os.Create(fmt.Sprintf("%s.pub.pem", filepath))
	if err != nil {
		return err
	}
	defer pubfile.Close()

	return pem.Encode(pubfile, block)
}
