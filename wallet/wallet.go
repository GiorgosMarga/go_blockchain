package wallet

import (
	"log"
	"time"
)

type Wallet struct {
}

func NewWallet() *Wallet {
	return &Wallet{}
}

func (w *Wallet) Start() {
	log.Println("Starting wallet application...")

	config := DummyConfig()

	core := NewCore(config, UtxoStore{})
	go core.start()

	time.Sleep(10 * time.Second)
	core.stop()
}
