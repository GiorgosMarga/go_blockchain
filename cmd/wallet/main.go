package main

import (
	"flag"
	"log"

	"github.com/GiorgosMarga/blockchain/wallet"
)

func main() {
	var (
		walletPort string
		serverPort string
		username   string
	)
	flag.StringVar(&walletPort, "wallet-port", ":9000", "Wallet listen address.")
	flag.StringVar(&serverPort, "server-port", ":9001", "Server listen address.")
	flag.StringVar(&username, "user", "Alice", "Dummy user to initialize wallet.")
	flag.Parse()
	var cfg wallet.Config
	if username == "Bob" {
		cfg = wallet.BobConfig()
	} else {
		cfg = wallet.AliceConfig()
	}
	wallet := wallet.NewWallet(walletPort, cfg)
	go wallet.Start()

	s := NewServer(serverPort, wallet, username)
	log.Fatal(s.Start())
}
