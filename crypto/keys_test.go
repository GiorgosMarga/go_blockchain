package crypto

import "testing"

func TestKeyGen(t *testing.T) {
	kp := NewKeyPair()
	if err := kp.LoadToFile("test_keypair"); err != nil {
		t.Fatal(err)
	}
}
