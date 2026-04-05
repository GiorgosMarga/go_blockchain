package miner

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GiorgosMarga/blockchain/block"
	"github.com/GiorgosMarga/blockchain/messages"
	"github.com/GiorgosMarga/blockchain/transport"
)

type MsgType byte

const (
	FetchTemplateResp MsgType = iota
)

type Miner struct {
	listenAddr string
	peersAddr  []string
	PublicKey  []byte
	isMining   atomic.Bool

	currTemplate    *block.Block
	currTemplateMtx *sync.Mutex

	blockChan chan *block.Block

	transport     transport.Transport
	internalChans map[MsgType]chan any
}

func New(address string, publicKey []byte, peers ...string) *Miner {
	m := &Miner{
		listenAddr:      address,
		peersAddr:       make([]string, 0, len(peers)),
		PublicKey:       publicKey,
		isMining:        atomic.Bool{},
		currTemplateMtx: &sync.Mutex{},
		blockChan:       make(chan *block.Block),
		transport:       transport.New(address),
		internalChans: map[MsgType]chan any{
			FetchTemplateResp: make(chan any, 10),
		},
	}

	for _, peerAddr := range peers {
		if err := m.transport.Connect(peerAddr); err != nil {
			fmt.Println(err)
			continue
		}
		m.peersAddr = append(m.peersAddr, peerAddr)
	}
	return m
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
		case tcpMsg := <-m.transport.Consume():
			switch msg := tcpMsg.(type) {
			case messages.Template:
				m.internalChans[FetchTemplateResp] <- msg
			default:
				fmt.Printf("invalid msg: %+v\n", msg)
			}
		case <-time.After(5 * time.Second):
			if m.isMining.Load() {
				go m.validateTemplate()
			} else {
				go m.fetchTemplate()
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
	if err := m.transport.Broadcast(submitBlock); err != nil {
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
	templateReq := messages.FetchTemplate{
		PublicKey: m.PublicKey,
		FromAddr:  m.listenAddr,
	}
	if err := m.transport.Broadcast(templateReq); err != nil {
		fmt.Println(err)
	}
	for {
		select {
		case msg := <-m.internalChans[FetchTemplateResp]:
			newTemplateMsg, ok := msg.(messages.Template)
			if !ok {
				fmt.Printf("[Miner]: received invalid template message: %+v\n", msg)
				continue
			}
			fmt.Printf("Received new template with target %x\n", newTemplateMsg.Block.Header.Target)
			m.currTemplateMtx.Lock()
			m.currTemplate = newTemplateMsg.Block
			m.currTemplateMtx.Unlock()
			m.isMining.Store(true)
			return
		case <-time.After(2 * time.Second):
			fmt.Printf("[Miner]: No new template\n")
			return
		}
	}
}
