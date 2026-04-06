package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"log"
	"time"

	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/messages"
	"github.com/GiorgosMarga/blockchain/transaction"
	"github.com/GiorgosMarga/blockchain/transport"
)

type MsgType byte

const (
	UtxosMsg MsgType = iota
)

type Core struct {
	Config        Config
	Utxos         UtxoStore
	TxChan        chan transaction.Transaction
	Transport     *transport.TCPTransport
	knownPeers    []string
	listenAddr    string
	internalChans map[MsgType]chan any
}

func NewCore(listenAddr string, config Config, utxos UtxoStore, peers ...string) *Core {
	c := &Core{
		listenAddr: listenAddr,
		Config:     config,
		Utxos:      utxos,
		TxChan:     make(chan transaction.Transaction),
		Transport:  transport.New(listenAddr),
		knownPeers: make([]string, 0, len(peers)),
		internalChans: map[MsgType]chan any{
			UtxosMsg: make(chan any, 10),
		},
	}

	for _, peerAddr := range peers {
		if err := c.Transport.Connect(peerAddr); err != nil {
			fmt.Println(err)
			continue
		}
		c.knownPeers = append(c.knownPeers, peerAddr)
	}
	return c
}
func (c *Core) start() error {
	go c.Transport.Start()

	go c.handleMessages()

	return c.FetchUtxos()
}
func (c *Core) stop() {
	c.stop()
}
func (c *Core) SendTx(tx *transaction.Transaction) error {
	msg := messages.SubmitTransaction{
		Tx: tx,
	}
	return c.Transport.Broadcast(msg)
}

func (c *Core) CreateTx(recipientKey ecdsa.PublicKey, amount uint) (*transaction.Transaction, error) {
	bufKey, _ := recipientKey.Bytes()
	log.Printf("Creating transaction for %d satoshis to %x\n", amount, bufKey)

	fee := c.calculateFee(amount)
	totalAmount := fee + amount

	inputs := make([]*transaction.TxInput, 0)
	inputSum := 0

	for pubKey, entries := range c.Utxos.utxos {
		pubBuf := elliptic.MarshalCompressed(elliptic.P256(), pubKey.X, pubKey.Y)
		for _, entry := range entries {
			if entry.IsSpent {
				continue
			}
			if inputSum >= int(totalAmount) {
				break
			}
			for _, kp := range c.Utxos.myKeys {
				if bytes.Equal(pubBuf, kp.PublicKeyBytes()) {
					sig, err := kp.Sign(entry.TxOutput.Hash())
					if err != nil {
						return nil, err
					}
					txInput := &transaction.TxInput{
						PrevTxOutputHash: entry.TxOutput.Hash(),
						Signature:        sig,
					}
					inputs = append(inputs, txInput)
					inputSum += int(entry.TxOutput.Value)
				}
			}
		}
		if inputSum >= int(totalAmount) {
			break
		}
	}
	if inputSum < int(totalAmount) {
		// log.Printf("Insufficient funds: have %d, need %d\n", inputSum, totalAmount)
		return nil, fmt.Errorf("Insufficient funds")
	}
	outputs := []*transaction.TxOutput{
		{
			Value:     uint64(amount),
			Id:        crypto.Random(),
			PublicKey: elliptic.MarshalCompressed(elliptic.P256(), recipientKey.X, recipientKey.Y),
		},
	}
	if inputSum > int(totalAmount) {
		outputs = append(outputs,
			&transaction.TxOutput{
				Value:     uint64(inputSum - int(amount)),
				Id:        crypto.Random(),
				PublicKey: c.Utxos.myKeys[0].PublicKeyBytes(),
			})
	}
	return &transaction.Transaction{
		Id:   crypto.Random(),
		Vin:  inputs,
		Vout: outputs,
	}, nil
}

func (c *Core) calculateFee(amount uint) uint {
	if c.Config.FeeConfig.FeeType == Fixed {
		return uint(c.Config.FeeConfig.Value)
	} else {
		return uint(float64(amount) * float64(c.Config.FeeConfig.Value) / 100)
	}
}

func (c *Core) GetBalance() uint64 {
	var balance uint64 = 0
	for _, entries := range c.Utxos.utxos {
		for _, entry := range entries {
			balance += entry.TxOutput.Value
		}
	}

	// log.Printf("Current balance is %d\n", balance)
	return balance
}

func (c *Core) FetchUtxos() error {
	// fmt.Printf("Trying to fetch utxos from default node %s...\n", c.Config.DefaultNode)

	for _, key := range c.Utxos.myKeys {
		msg := messages.FetchUTXOsReq{
			PublicKey: key.PublicKeyBytes(),
			FromAddr:  c.listenAddr,
		}
		if err := c.Transport.Send(c.Config.DefaultNode, msg); err != nil {
			log.Println(err)
		}
		select {
		case msg := <-c.internalChans[UtxosMsg]:
			utxoMsg, ok := msg.(messages.UTXOsResp)
			if !ok {
				fmt.Printf("[Wallet]: received invalid msg %+v\n", utxoMsg)
				continue
			}
			c.Utxos.utxos[key.PublicKey] = utxoMsg.Utxos
		case <-time.After(2 * time.Second):
			continue
		}
	}
	return nil
}
func (c *Core) handleMessages() {
	for tcpMsg := range c.Transport.Consume() {
		switch msg := tcpMsg.(type) {
		case messages.UTXOsResp:
			c.internalChans[UtxosMsg] <- msg
		}
	}
}
