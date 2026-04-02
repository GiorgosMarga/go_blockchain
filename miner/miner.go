package miner

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GiorgosMarga/blockchain/block"
	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/messages"
	"github.com/GiorgosMarga/blockchain/params"
	"github.com/GiorgosMarga/blockchain/transaction"
	"github.com/GiorgosMarga/blockchain/transport"
)

type Miner struct {
	PublicKey crypto.Hash
	isMining  atomic.Bool

	currTemplate    *block.Block
	currTemplateMtx *sync.Mutex

	blockChan chan *block.Block

	transport transport.Transport
}

func New(address string, publicKey crypto.Hash) *Miner {
	return &Miner{
		PublicKey:       publicKey,
		isMining:        atomic.Bool{},
		currTemplateMtx: &sync.Mutex{},
		blockChan:       make(chan *block.Block),
		transport:       transport.New(address),
	}
}

func (m *Miner) Start() error {
	go func() {
		if err := m.transport.Start(); err != nil {
			fmt.Println(err)
		}
	}()
	go m.startMining()

	for {
		select {
		case b := <-m.blockChan:
			log.Printf("[Miner]: Mined new block %x\n", b.Hash())
			m.submitBlock()
		case <-time.After(5 * time.Second):
			if m.isMining.Load() {
				m.validateTemplate()
			} else {
				m.fetchTemplate()
			}
		}
	}
}

func (m *Miner) startMining() {
	for {
		if m.isMining.Load() {
			m.currTemplateMtx.Lock()
			// log.Printf("[Miner]: Mining block with target %x\n", m.currTemplate.Header.Target)
			if m.currTemplate.Mine(2_000_000) {
				// log.Printf("[Miner]: Block mined: %x", m.currTemplate.Hash())
				m.isMining.Store(false)
				m.blockChan <- m.currTemplate
			}
			m.currTemplateMtx.Unlock()
		}
	}
}
func (m *Miner) submitBlock() {
	log.Println("Submiting block...")

	submitBlock := messages.SubmitTemplate{
		Block: m.currTemplate,
	}

	buf := new(bytes.Buffer)

	if err := gob.NewEncoder(buf).Encode(submitBlock); err != nil {
		panic(err)
	}

	if err := m.transport.Broadcast(buf.Bytes()); err != nil {
		fmt.Println(err)
	}
}
func (m *Miner) validateTemplate() {
	log.Println("Validating template...")

	// valTemplate := messages.ValidateTemplate{
	// 	Block: m.currTemplate,
	// }

	// buf := new(bytes.Buffer)

	// if err := gob.NewEncoder(buf).Encode(	valTemplate ); err != nil {
	// 	panic(err)
	// }

	// if err := m.transport.Broadcast(buf.Bytes()); err != nil {
	// 	fmt.Println(err)
	// }
}
func (m *Miner) fetchTemplate() {
	log.Println("Fetching template...")
	// TODO: send real msg
	// templateReq := messages.FetchTemplate{
	// 	PublicKey: m.PublicKey,
	// }
	// buf := new(bytes.Buffer)
	// if err := gob.NewEncoder(buf).Encode(templateReq); err != nil {
	// 	panic(err)
	// }
	// if err := m.transport.Broadcast(buf.Bytes()); err != nil {
	// 	fmt.Println(err)
	// }

	testTemplate := &block.Block{
		Header: block.Header{
			PrevBlockHash: crypto.Random(),
			MerkleRoot:    crypto.Random(),
			Target:        params.MyConfig.MinTarget,
			Difficulty:    1,
		},
		Txs: make([]*transaction.Transaction, 1),
	}
	testTemplate.Txs[0] = &transaction.Transaction{
		Id:  crypto.Zero(),
		Vin: []*transaction.TxInput{},
		Vout: []*transaction.TxOutput{
			{
				Value:     10,
				Id:        fmt.Sprintf("%x", crypto.Random()),
				PublicKey: crypto.Random(),
			},
			{
				Value:     12,
				Id:        fmt.Sprintf("%x", crypto.Random()),
				PublicKey: crypto.Random(),
			},
		},
	}
	fmt.Printf("Received new template with target %x\n", testTemplate.Header.Target)
	m.currTemplateMtx.Lock()
	m.currTemplate = testTemplate
	m.currTemplateMtx.Unlock()
	m.isMining.Store(true)
}
