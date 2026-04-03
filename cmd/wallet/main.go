package main

import "github.com/GiorgosMarga/blockchain/wallet"

func main() {
	wallet := wallet.NewWallet()
	wallet.Start()

}
