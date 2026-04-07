package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/miner"
)

func main() {
	var (
		keyPath    string
		port       string
		knownPeers []string
	)
	flag.StringVar(&keyPath, "public_key_path", "alice.pub.pem", "Public key path name.")
	flag.StringVar(&port, "port", ":6000", "Miner address.")
	flag.Func("peers", "Known peer addresses seperated with (,).", func(s string) error {
		knownPeers = strings.Split(s, ",")
		return nil
	})
	flag.Parse()

	kp, err := crypto.LoadPubliFromFile(keyPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Public key %x\n", kp.PublicKeyBytes())
	m := miner.New(port, kp.PublicKeyBytes(), knownPeers...)
	m.Start()
}
