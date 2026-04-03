package main

import (
	"flag"
	"log"

	"github.com/GiorgosMarga/blockchain/crypto"
)

func main() {
	var filepath string
	flag.StringVar(&filepath, "fp", "", "A filepath to store the new key pair.")
	flag.Parse()
	kp := crypto.NewKeyPair()

	if err := kp.LoadToFile(filepath); err != nil {
		log.Fatal(err)
	}

}
