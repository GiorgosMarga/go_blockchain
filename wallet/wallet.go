package wallet

import (
	"crypto/ecdsa"
	"crypto/elliptic"

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
				MyKeys: cfg.MyKeys,
				Utxos:  make(map[ecdsa.PublicKey][]utils.UtxoEntry),
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

func (w *Wallet) Send(address []byte, amount uint) error {
	pubKey, err := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), address)
	if err != nil {
		return err
	}
	newTx, err := w.Core.CreateTx(*pubKey, amount)
	if err != nil {
		return err
	}
	if err := w.Core.SendTx(newTx); err != nil {
		return err
	}
	return nil
}
