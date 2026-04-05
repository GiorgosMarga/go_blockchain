package main

import (
	"flag"

	"github.com/GiorgosMarga/blockchain/wallet"
)

func main() {

	var listenAddr string
	flag.StringVar(&listenAddr, "port", ":9000", "Wallet listen address.")
	flag.Parse()

	wallet := wallet.NewWallet(listenAddr)
	wallet.Start()

}
