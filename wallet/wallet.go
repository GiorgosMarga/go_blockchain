package wallet

import (
	"crypto/ecdsa"
	"log"
	"time"

	"github.com/GiorgosMarga/blockchain/utils"
)

type Wallet struct {
	listenAddr string
}

func NewWallet(listenAddr string) *Wallet {
	return &Wallet{
		listenAddr: listenAddr,
	}
}

func (w *Wallet) Start() {
	log.Println("Starting wallet application...")

	config := DummyConfig()
	core := NewCore(
		w.listenAddr,
		config,
		UtxoStore{
			myKeys: config.MyKeys,
			utxos:  make(map[ecdsa.PublicKey][]utils.UtxoEntry),
		}, config.DefaultNode)
	go core.start()


	time.Sleep(10 * time.Second)
	newTx, err := core.CreateTx(config.Contacts[0].PubKey, 1_000)
	if err != nil {
		panic(err)
	}

	if err := core.SendTx(newTx); err != nil {
		panic(err)
	}
	time.Sleep(10 * time.Minute)
	core.stop()
}
