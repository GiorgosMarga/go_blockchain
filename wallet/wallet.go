package wallet

import (
	"crypto/ecdsa"

	"github.com/GiorgosMarga/blockchain/utils"
)

type Wallet struct {
	Core     *Core
	quitChan chan struct{}
}

func NewWallet(listenAddr string, cfg Config) *Wallet {
	return &Wallet{
		quitChan: make(chan struct{}),
		Core: NewCore(
			listenAddr,
			cfg,
			UtxoStore{
				myKeys: cfg.MyKeys,
				utxos:  make(map[ecdsa.PublicKey][]utils.UtxoEntry),
			}, cfg.DefaultNode),
	}
}

func (w *Wallet) Start() {
	if err := w.Core.start(); err != nil {
		panic(err)
	}
	<-w.quitChan
	w.Core.stop()
}
func (w *Wallet) UpdateBalance() error {
	return w.Core.FetchUtxos()
}
func (w *Wallet) Balance() uint64 {
	return w.Core.GetBalance()
}
func (w *Wallet) Stop() {
	w.quitChan <- struct{}{}
}

func (w *Wallet) CreateTx(recipientPubKey ecdsa.PublicKey, amount uint) error {
	newTx, err := w.Core.CreateTx(recipientPubKey, amount)
	if err != nil {
		return err
	}

	if err := w.Core.SendTx(newTx); err != nil {
		return err
	}
	return nil
}
