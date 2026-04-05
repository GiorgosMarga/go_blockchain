package node

import (
	"fmt"
	"math"
	"os"
	"reflect"
	"slices"
	"time"

	"github.com/GiorgosMarga/blockchain/block"
	"github.com/GiorgosMarga/blockchain/blockchain"
	"github.com/GiorgosMarga/blockchain/crypto"
	"github.com/GiorgosMarga/blockchain/messages"
	"github.com/GiorgosMarga/blockchain/params"
	"github.com/GiorgosMarga/blockchain/transaction"
	"github.com/GiorgosMarga/blockchain/transport"
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
		Blockchain: blockchain.New(params.MyConfig),
	}
	go n.Transport.Start()

	n.internalChans = map[MsgChan]chan any{
		BlockResp:      make(chan any, 5),
		DifferenceResp: make(chan any, 5),
	}
	for _, peerNode := range peerNodes {
		if err := n.Transport.Connect(peerNode); err != nil {
			fmt.Printf("[Node]: error connecting with %s: %s\n", peerNode, err)
		}
		n.peerNodes = append(n.peerNodes, peerNode)
	}

	return n
}

func (n *Node) Start() {
	go n.saveBlockchain()
	go n.handleMessages()
	if fileExists(n.bcpath) {
		fmt.Println("[Node]: loading blockchain from file...")
		if err := n.loadBlockchain(n.bcpath); err != nil {
			panic(err)
		}
	} else {
		if len(n.peerNodes) == 0 {
			// seed node
			fmt.Println("[Node]: seed node...")
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

	time.Sleep(100 * time.Minute)

}
func (n *Node) handleMessages() {
	for receivedMsg := range n.Transport.Consume() {
		switch msg := receivedMsg.(type) {
		case messages.DifferenceReq:
			fmt.Println("difference request")
			if err := n.handleDifferenceReq(msg); err != nil {
				fmt.Println(err)
			}
		case messages.DifferenceResp:
			n.internalChans[DifferenceResp] <- msg
		case messages.FetchBlockReq:
			if err := n.handleBlockReq(msg); err != nil {
				fmt.Println(err)
			}
		case messages.FetchBlockResp:
			n.internalChans[BlockResp] <- msg
		case messages.FetchUTXOsReq:
			if err := n.handleFetchUtxos(msg); err != nil {
				fmt.Println(err)
			}
		case messages.NewBlock:
			if err := n.handleNewBlock(msg); err != nil {
				fmt.Println(err)
			}
		case messages.NewTx:
			if err := n.handleNewTx(msg); err != nil {
				fmt.Println(err)
			}
		case messages.ValidateTemplateReq:
			if err := n.handleValidateTemplateReq(msg); err != nil {
				fmt.Println(err)
			}
		case messages.SubmitTemplate:
			if err := n.handleSubmitTemplate(msg); err != nil {
				fmt.Println(err)
			}
		case messages.SubmitTransaction:
			if err := n.handleSubmitTx(msg); err != nil {
				fmt.Println(err)
			}
		case messages.FetchTemplate:
			if err := n.handleFetchTemplate(msg); err != nil {
				fmt.Println(err)
			}
		default:
			fmt.Printf("[Node]: Received invalid msg: %+v, %s\n", msg, reflect.TypeOf(msg))
		}
	}
}
func (n *Node) getBlockchain(fromAddr string, numOfBlocks int) error {
	fmt.Printf("[Node]: Fetching blockchain from %s\n", fromAddr)
	for i := range numOfBlocks {
		msg := messages.FetchBlockReq{Height: i, FromAddr: n.listenAddr}
		if err := n.Transport.Send(fromAddr, msg); err != nil {
			return err
		}
		for resp := range n.internalChans[BlockResp] {
			blockResp, ok := resp.(messages.FetchBlockResp)
			if !ok {
				fmt.Printf("[Node]: received invalid block response for height %d\n", i)
				continue
			}
			// previous block
			if blockResp.Height != i {
				continue
			}
			if err := n.Blockchain.AddBlock(blockResp.Block); err != nil {
				fmt.Printf("[Node]: error adding block with height %d: %s\n", i, err)
				continue
			}
			fmt.Printf("[Node]: added block with height: %d\n", i)
			break
		}
	}
	fmt.Printf("[Node]: Received %d/%d blocks\n", len(n.Blockchain.Blocks), numOfBlocks)
	return nil
}
func (n *Node) findLongestChainNode() (string, int, error) {
	maxHeight := math.MinInt
	maxHeightPeer := ""

	msg := messages.DifferenceReq{Height: 0, FromAddr: n.listenAddr}
	for _, peer := range n.peerNodes {
		fmt.Printf("[Node]: requesting height from peer %s\n", peer)
		if err := n.Transport.Send(peer, msg); err != nil {
			return "", -1, err
		}
	respLoop:
		for {
			select {
			case resp := <-n.internalChans[DifferenceResp]:
				msg, ok := resp.(messages.DifferenceResp)
				if !ok {
					fmt.Printf("[Node]: peer %s sent invalid difference response\n", peer)
					continue
				}
				if msg.Height > maxHeight {
					maxHeight = msg.Height
					maxHeightPeer = peer
				}
				break respLoop
			case <-time.After(20 * time.Second):
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
			fmt.Printf("[Node]: failed to save blockchain: %s\n", err)
		}
	}
}

func (n *Node) handleBlockReq(blockReq messages.FetchBlockReq) error {
	if blockReq.Height > len(n.Blockchain.Blocks) {
		return nil
	}
	block := n.Blockchain.Blocks[blockReq.Height]
	resp := messages.FetchBlockResp{
		Block:    block,
		FromAddr: n.listenAddr,
		Height:   blockReq.Height,
	}
	return n.Transport.Send(blockReq.FromAddr, resp)
}
func (n *Node) handleDifferenceReq(diffReq messages.DifferenceReq) error {
	resp := messages.DifferenceResp{
		Height:   len(n.Blockchain.Blocks) - diffReq.Height,
		FromAddr: n.listenAddr,
	}
	return n.Transport.Send(diffReq.FromAddr, resp)
}

func (n *Node) handleFetchUtxos(utxoReq messages.FetchUTXOsReq) error {
	// x, y := elliptic.UnmarshalCompressed(elliptic.P256(), utxoReq.PublicKey)

	// pubKey := ecdsa.PublicKey{
	// 	Curve: elliptic.P256(),
	// 	X:     x,
	// 	Y:     y,
	// }

	msg := messages.UTXOsResp{
		Utxos:    n.Blockchain.GetUtxos(utxoReq.PublicKey),
		FromAddr: n.listenAddr,
	}
	return n.Transport.Send(utxoReq.FromAddr, msg)
}
func (n *Node) handleNewBlock(newBlockMsg messages.NewBlock) error {
	return n.Blockchain.AddBlock(newBlockMsg.Block)
}
func (n *Node) handleNewTx(newTxMsg messages.NewTx) error {
	return n.Blockchain.AddToMempool(newTxMsg.Tx)
}
func (n *Node) handleValidateTemplateReq(validateTemplateReq messages.ValidateTemplateReq) error {
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
	return n.Transport.Send(validateTemplateReq.FromAddr, msg)
}
func (n *Node) handleSubmitTemplate(template messages.SubmitTemplate) error {
	block := template.Block
	if err := n.Blockchain.AddBlock(block); err != nil {
		return err
	}
	fmt.Printf("[Node]: New block was added with hash: %x\n", block.BlockHash)

	n.Blockchain.RebuildUtxos()
	// bcast new block
	newBlockMsg := messages.NewBlock{
		Block: block,
	}

	return n.Transport.Broadcast(newBlockMsg)
}
func (n *Node) handleSubmitTx(txMsg messages.SubmitTransaction) error {
	tx := txMsg.Tx
	if err := n.Blockchain.AddToMempool(tx); err != nil {
		return err
	}

	n.Blockchain.RebuildUtxos()
	// bcast new tx
	newTxMsg := messages.NewTx{
		Tx: tx,
	}

	return n.Transport.Broadcast(newTxMsg)
}

// TODO: fix merkle root and coinbase tx
func (n *Node) handleFetchTemplate(msg messages.FetchTemplate) error {
	fmt.Printf("Fetch template msg\n")

	prevBlockHash := crypto.Zero()
	if len(n.Blockchain.Blocks) > 0 {
		prevBlockHash = n.Blockchain.Blocks[len(n.Blockchain.Blocks)-1].Hash()
	}
	txs := n.Blockchain.GetTxsFromMempool()
	if len(txs) == 0 {
		return nil
	}
	coinbaseTx := &transaction.Transaction{
		Id:  crypto.Random(),
		Vin: []*transaction.TxInput{},
		Vout: []*transaction.TxOutput{
			{
				Value:     0,
				Id:        crypto.Random(),
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
	block.BlockHash = block.Hash()

	templateMsg := messages.Template{
		Block: &block,
	}
	return n.Transport.Send(msg.FromAddr, templateMsg)
}
func fileExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	return os.IsExist(err)
}
