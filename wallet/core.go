package wallet

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/gob"
	"fmt"
	"log"

	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/messages"
	"github.com/GiorgosMarga/blockchain/transaction"
	"github.com/GiorgosMarga/blockchain/transport"
)

type Core struct {
	Config    Config
	Utxos     UtxoStore
	TxChan    chan transaction.Transaction
	Transport *transport.TCPTransport
}

func NewCore(config Config, utxos UtxoStore) *Core {
	return &Core{
		Config:    config,
		Utxos:     utxos,
		TxChan:    make(chan transaction.Transaction),
		Transport: transport.New(":3001"),
	}
}
func (c *Core) start() {
	go c.Transport.Start()
	c.handleMessages()
}
func (c *Core) stop() {
	c.stop()
}
func (c *Core) SendTx(tx *transaction.Transaction) error {
	msg := messages.SubmitTransaction{
		Tx: tx,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}
	return c.Transport.Broadcast(buf.Bytes())
}

func (c *Core) CreateTx(recipientKey ecdsa.PublicKey, amount uint) (*transaction.Transaction, error) {
	bufKey, _ := recipientKey.Bytes()
	log.Printf("Creating transaction for %d satoshis to %x\n", amount, bufKey)

	fee := c.calculateFee(amount)
	totalAmount := fee + amount

	inputs := make([]*transaction.TxInput, 0)
	inputSum := 0

	for pubKey, entries := range c.Utxos.utxos {
		for _, entry := range entries {
			if entry.IsSpent {
				continue
			}
			if inputSum >= int(totalAmount) {
				break
			}
			for _, kp := range c.Utxos.myKeys {
				if pubKey.Equal(kp.PublicKey) {
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
		log.Printf("Insufficient funds: have %d, need %d\n", inputSum, totalAmount)
		return nil, fmt.Errorf("Insufficient funds")
	}
	outputs := []*transaction.TxOutput{
		{
			Value:     uint64(amount),
			Id:        "unique_id_",
			PublicKey: crypto.Hash(bufKey[:]),
		},
	}
	if inputSum > int(totalAmount) {
		myPubKey, _ := c.Utxos.myKeys[0].PublicKey.Bytes()
		outputs = append(outputs,
			&transaction.TxOutput{
				Value:     uint64(inputSum - int(amount)),
				Id:        "unique_id_2",
				PublicKey: crypto.Hash(myPubKey[:]),
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

	log.Printf("Current balance is %d\n", balance)
	return balance
}

func (c *Core) FetchUtxos() error {
	log.Printf("Trying to fetch utxos from default node %s...\n", c.Config.DefaultNode)

	for _, key := range c.Utxos.myKeys {
		msg := messages.FetchUTXOsReq{
			PublicKey: key.PublicKey,
		}
		buf := new(bytes.Buffer)
		if err := gob.NewEncoder(buf).Encode(msg); err != nil {
			return err
		}
		if err := c.Transport.Send(c.Config.DefaultNode, buf.Bytes()); err != nil {
			log.Println(err)
		}
	}

	return nil
}
func (c *Core) handleMessages() {
	for msg := range c.Transport.Consume() {
		utxoResp := messages.UTXOsResp{}
		gob.NewDecoder(bytes.NewReader(msg)).Decode(&utxoResp)
		fmt.Printf("Received utxo msg %+v\n", utxoResp)
	}
	fmt.Println("stopping")
}
