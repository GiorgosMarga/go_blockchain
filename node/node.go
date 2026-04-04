package node

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"slices"
	"time"

	"github.com/GiorgosMarga/blockchain/block"
	"github.com/GiorgosMarga/blockchain/blockchain"
	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/messages"
	"github.com/GiorgosMarga/blockchain/params"
	"github.com/GiorgosMarga/blockchain/transaction"
	"github.com/GiorgosMarga/blockchain/transport"
	"github.com/GiorgosMarga/blockchain/utils"
)

type MsgChan byte

const (
	DifferenceResp MsgChan = iota
	BlockResp
)

type Node struct {
	listenAddr    string
	bcpath        string // load blockchain from this file if exists
	peerNodes     []string
	Transport     transport.Transport
	Blockchain    *blockchain.Blockchain
	internalChans map[MsgChan]chan any
}

func (n *Node) loadBlockchain(filepath string) error {
	if err := n.Blockchain.LoadFromFile(filepath); err != nil {
		return err
	}
	// rebuild utxos
	n.Blockchain.RebuildUtxos()
	// try to adjust target
	n.Blockchain.TryAdjustTarget()
	return nil
}

func NewNode(listenAddr, blockchainPath string, peerNodes ...string) *Node {
	n := &Node{
		listenAddr: listenAddr,
		Transport:  transport.New(listenAddr),
		bcpath:     blockchainPath,
		peerNodes:  make([]string, 0, len(peerNodes)),
		Blockchain: &blockchain.Blockchain{},
	}
	go n.Transport.Start()

	n.internalChans = map[MsgChan]chan any{
		BlockResp:      make(chan any, 5),
		DifferenceResp: make(chan any, 5),
	}
	for _, peerNode := range peerNodes {
		if err := n.Transport.Connect(peerNode); err != nil {
			log.Printf("[Node]: error connecting with %s: %s\n", peerNode, err)
		}
		n.peerNodes = append(n.peerNodes, peerNode)
	}
	if fileExists(n.bcpath) {
		if err := n.loadBlockchain(n.bcpath); err != nil {
			panic(err)
		}
	} else {
		if len(peerNodes) == 0 {
			// seed node
			fmt.Println("seed node")
			n.Blockchain = blockchain.New(params.MyConfig)
		} else {
			maxHeightPeer, maxHeight, err := n.findLongestChainNode()
			if err != nil {
				panic(err)
			}
			fmt.Println("Max height peer:", maxHeightPeer, "max height:", maxHeight)
			if err := n.getBlockchain(maxHeightPeer, maxHeight); err != nil {
				panic(err)
			}
			n.Blockchain.RebuildUtxos()
			n.Blockchain.TryAdjustTarget()
		}
	}

	return n
}

func (n *Node) Start() {
	go n.saveBlockchain()
	for msg := range n.Transport.Consume() {
		b := bytes.NewReader(msg)
		var receivedMsg any
		if err := gob.NewDecoder(b).Decode(&receivedMsg); err != nil {
			log.Println(err)
			continue
		}
		switch msg := receivedMsg.(type) {
		case *messages.DifferenceReq:
			fmt.Println("difference request")
			if err := n.handleDifferenceReq(msg); err != nil {
				log.Println(err)
			}
		case *messages.DifferenceResp:
			n.internalChans[DifferenceResp] <- receivedMsg
		case *messages.FetchBlockReq:
			if err := n.handleBlockReq(msg); err != nil {
				log.Println(err)
			}
		case *messages.FetchBlockResp:
			n.internalChans[BlockResp] <- receivedMsg
		case *messages.FetchUTXOsReq:
			if err := n.handleFetchUtxos(msg); err != nil {
				log.Println(err)
			}
		case *messages.NewBlock:
			if err := n.handleNewBlock(msg); err != nil {
				log.Println(err)
			}
		case *messages.NewTx:
			if err := n.handleNewTx(msg); err != nil {
				log.Println(err)
			}
		case *messages.ValidateTemplateReq:
			if err := n.handleValidateTemplateReq(msg); err != nil {
				log.Println(err)
			}
		case *messages.SubmitTemplate:
			if err := n.handleSubmitTemplate(msg); err != nil {
				log.Println(err)
			}
		case *messages.SubmitTransaction:
			if err := n.handleSubmitTx(msg); err != nil {
				log.Println(err)
			}
		case *messages.FetchTemplate:
			if err := n.handleFetchTemplate(msg); err != nil {
				log.Println(err)
			}
		default:
			log.Printf("[Node]: Received invalid msg: %+v\n", msg)
		}
	}
}

func (n *Node) getBlockchain(fromAddr string, numOfBlocks int) error {
	for i := range numOfBlocks {
		msg := messages.FetchBlockReq{Height: i}
		buf := new(bytes.Buffer)
		if err := gob.NewEncoder(buf).Encode(msg); err != nil {
			return err
		}
		if err := n.Transport.Send(fromAddr, buf.Bytes()); err != nil {
			return err
		}
		for resp := range n.internalChans[BlockResp] {
			blockResp, ok := resp.(*messages.FetchBlockResp)
			if !ok {
				log.Printf("[Node]: received invalid block response for height %d\n", i)
				continue
			}
			// previous block
			if blockResp.Height != i {
				continue
			}
			if err := n.Blockchain.AddBlock(blockResp.Block); err != nil {
				log.Printf("[Node]: error adding block with height %d: %s\n", i, err)
				continue
			}
			break
		}
	}
	return nil
}
func (n *Node) findLongestChainNode() (string, int, error) {
	maxHeight := 0
	maxHeightPeer := ""

	msg := &messages.DifferenceReq{Height: 0}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		log.Printf("[Node]: error encoding difference request message: %s\n", err)
		return "", -1, err
	}
	for _, peer := range n.peerNodes {
		log.Printf("[Node]: requesting height from peer %s\n", peer)
		if err := n.Transport.Send(peer, buf.Bytes()); err != nil {
			return "", -1, err
		}
	respLoop:
		for {
			select {
			case resp := <-n.internalChans[DifferenceResp]:
				msg, ok := resp.(*messages.DifferenceResp)
				if !ok {
					log.Printf("[Node]: peer %s sent invalid difference response\n", peer)
					continue
				}
				if msg.Height > maxHeight {
					maxHeight = msg.Height
					maxHeightPeer = peer
				}
				break respLoop
			case <-time.After(2 * time.Second):
				break respLoop
			}
		}
	}
	return maxHeightPeer, maxHeight, nil
}

func (n *Node) saveBlockchain() {
	interval := time.NewTicker(30 * time.Second)
	for range interval.C {
		fmt.Println("[Node]: saving blockchain to disk...")
		if err := n.Blockchain.LoadToFile(n.bcpath); err != nil {
			log.Printf("[Node]: failed to save blockchain: %s\n", err)
		}
	}
}

func (n *Node) handleBlockReq(blockReq *messages.FetchBlockReq) error {
	if blockReq.Height > len(n.Blockchain.Blocks) {
		return nil
	}
	block := n.Blockchain.Blocks[blockReq.Height]
	resp := &messages.FetchBlockResp{
		Block:    block,
		FromAddr: n.listenAddr,
		Height:   blockReq.Height,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(resp); err != nil {
		return err
	}
	return n.Transport.Send(blockReq.FromAddr, buf.Bytes())
}
func (n *Node) handleDifferenceReq(diffReq *messages.DifferenceReq) error {
	resp := &messages.DifferenceResp{
		Height:   len(n.Blockchain.Blocks) - diffReq.Height,
		FromAddr: n.listenAddr,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(resp); err != nil {
		return err
	}
	return n.Transport.Send(diffReq.FromAddr, buf.Bytes())
}

func (n *Node) handleFetchUtxos(utxoReq *messages.FetchUTXOsReq) error {
	utxos := make([]utils.UtxoEntry, 0)
	msg := &messages.UTXOsResp{
		Utxos:    utxos,
		FromAddr: n.listenAddr,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}
	return n.Transport.Send(utxoReq.FromAddr, buf.Bytes())
}
func (n *Node) handleNewBlock(newBlockMsg *messages.NewBlock) error {
	return n.Blockchain.AddBlock(newBlockMsg.Block)
}
func (n *Node) handleNewTx(newTxMsg *messages.NewTx) error {
	return n.Blockchain.AddToMempool(newTxMsg.Tx)
}
func (n *Node) handleValidateTemplateReq(validateTemplateReq *messages.ValidateTemplateReq) error {
	block := validateTemplateReq.Block
	var isValid bool
	if len(n.Blockchain.Blocks) == 0 {
		isValid = block.Header.PrevBlockHash == crypto.Zero()
	} else {
		isValid = block.Header.PrevBlockHash == n.Blockchain.Blocks[len(n.Blockchain.Blocks)-1].Hash()
	}
	msg := messages.ValidateTemplateResp{
		IsValid: isValid,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil {
		return err
	}
	return n.Transport.Send(validateTemplateReq.FromAddr, buf.Bytes())
}
func (n *Node) handleSubmitTemplate(template *messages.SubmitTemplate) error {
	block := template.Block
	if err := n.Blockchain.AddBlock(block); err != nil {
		return err
	}

	n.Blockchain.RebuildUtxos()
	// bcast new block
	newBlockMsg := messages.NewBlock{
		Block: block,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(newBlockMsg); err != nil {
		return err
	}

	return n.Transport.Broadcast(buf.Bytes())
}
func (n *Node) handleSubmitTx(txMsg *messages.SubmitTransaction) error {
	tx := txMsg.Tx
	if err := n.Blockchain.AddToMempool(tx); err != nil {
		return err
	}

	n.Blockchain.RebuildUtxos()
	// bcast new tx
	newTxMsg := messages.NewTx{
		Tx: tx,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(newTxMsg); err != nil {
		return err
	}

	return n.Transport.Broadcast(buf.Bytes())
}

// TODO: fix merkle root and coinbase tx
func (n *Node) handleFetchTemplate(msg *messages.FetchTemplate) error {
	prevBlockHash := crypto.Zero()
	if len(n.Blockchain.Blocks) > 0 {
		prevBlockHash = n.Blockchain.Blocks[len(n.Blockchain.Blocks)-1].Hash()
	}
	txs := n.Blockchain.GetTxsFromMempool()
	coinbaseTx := &transaction.Transaction{
		Id:  crypto.Random(),
		Vin: []*transaction.TxInput{},
		Vout: []*transaction.TxOutput{
			{
				Value:     0,
				Id:        "random_id",
				PublicKey: msg.PublicKey,
			},
		},
	}
	txs = slices.Insert(txs, 0, coinbaseTx)
	merkleRoot := crypto.CalculateMerkleRoot(txs)
	block := block.Block{
		Header: block.Header{
			Timestamp:     time.Now().UnixMilli(),
			PrevBlockHash: prevBlockHash,
			MerkleRoot:    merkleRoot,
			Nonce:         0,
			Target:        n.Blockchain.Target,
		},
		Txs: txs,
	}
	minerFees, err := block.CalculateMinerFees(n.Blockchain.Utxos)
	if err != nil {
		panic(err)
	}
	reward := n.Blockchain.CalcBlockReward()
	block.Txs[0].Vout[0].Value =
		reward + minerFees
	block.Header.MerkleRoot =
		crypto.CalculateMerkleRoot(block.Txs)

	templateMsg := &messages.Template{
		Block: &block,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(templateMsg); err != nil {
		return err
	}
	return n.Transport.Send(msg.FromAddr, buf.Bytes())
}
func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}
