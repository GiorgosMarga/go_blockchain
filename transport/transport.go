package transport

import (
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"

	"github.com/GiorgosMarga/blockchain/messages"
)

var (
	ErrPeerNotFound = errors.New("peer not found")
)

type Transport interface {
	Start() error
	Stop()
	Send(string, any) error
	Broadcast(any) error
	Consume() <-chan any
	AddPeer(string)
	Connect(string) error
}
type TCPMessage struct {
	From    string
	To      string
	Payload any
}
type Peer struct {
	conn       net.Conn
	decoder    *gob.Decoder
	encoder    *gob.Encoder
	address    string
	id         uint32
	isOutbound bool
}

type TCPTransport struct {
	Address     string
	Id          uint32
	ln          net.Listener
	stopChan    chan struct{}
	outputChan  chan any
	peers       map[string]*Peer
	HandshakeFn func(net.Conn, string, uint32) (*Peer, error)
	mtx         *sync.Mutex
}

func DefaultHandshakeFn(conn net.Conn, myAddr string, myId uint32) (*Peer, error) {
	handshakeMsg := make([]byte, 0)
	handshakeMsg = binary.LittleEndian.AppendUint32(handshakeMsg, myId)
	handshakeMsg = append(handshakeMsg, []byte(myAddr)...)

	if _, err := conn.Write(handshakeMsg); err != nil {
		return nil, err
	}
	handshakeMsgResp := make([]byte, 1024)
	n, err := conn.Read(handshakeMsgResp)
	if err != nil {
		return nil, err
	}
	peer := &Peer{}
	peer.conn = conn
	peer.id = binary.LittleEndian.Uint32(handshakeMsgResp[:n])
	peer.address = string(handshakeMsgResp[4:n])
	peer.decoder = gob.NewDecoder(conn)
	peer.encoder = gob.NewEncoder(conn)
	return peer, nil
}
func New(address string) *TCPTransport {
	gobInit()
	return &TCPTransport{
		Address:     address,
		stopChan:    make(chan struct{}),
		outputChan:  make(chan any, 10),
		peers:       make(map[string]*Peer),
		Id:          rand.Uint32(),
		HandshakeFn: DefaultHandshakeFn,
		mtx:         &sync.Mutex{},
	}
}

func gobInit() {
	gob.Register(messages.DifferenceReq{})
	gob.Register(messages.DifferenceResp{})
	gob.Register(messages.FetchBlockReq{})
	gob.Register(messages.FetchBlockResp{})
	gob.Register(messages.FetchTemplate{})
	gob.Register(messages.Template{})
	gob.Register(messages.FetchUTXOsReq{})
	gob.Register(messages.UTXOsResp{})
	gob.Register(messages.SubmitTransaction{})
	gob.Register(messages.NewTx{})
	gob.Register(messages.NewBlock{})
	gob.Register(messages.SubmitTemplate{})
}

func (t *TCPTransport) AddPeer(addr string) {
	t.peers[addr] = &Peer{
		address: addr,
	}
}
func (t *TCPTransport) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	peer, err := t.HandshakeFn(conn, t.Address, t.Id)
	if err != nil {
		conn.Close()
		return err
	}
	peer.isOutbound = true
	if !t.registerPeer(peer) {
		// rejected (duplicate)
		conn.Close()
		return nil
	}
	go t.readLoop(peer)
	return nil
}

func (t *TCPTransport) registerPeer(peer *Peer) bool {
	t.mtx.Lock()
	defer t.mtx.Unlock()

	existingPeer, exists := t.peers[peer.address]
	if !exists {
		t.peers[peer.address] = peer
		return true
	}

	// peer exists need to keep only one (inbound or outbound)
	// prefer outbound
	if existingPeer.isOutbound {
		return false
	}
	if peer.isOutbound {
		existingPeer.conn.Close()
		t.peers[peer.address] = peer
		return true
	}
	// fallback deterministic rule
	if t.Id < peer.id {
		return false
	}
	existingPeer.conn.Close()
	t.peers[peer.address] = peer
	return true
}
func (t *TCPTransport) Start() error {
	ln, err := net.Listen("tcp", t.Address)
	if err != nil {
		return err
	}
	t.ln = ln
	// log.Printf("[TCP]: Listening on port: %s...\n", t.Address)
	for {
		select {
		// TODO: shutdown all handle conn loops
		case <-t.stopChan:
			fmt.Println("Stopping transport...")
		default:
			conn, err := ln.Accept()
			if err != nil {
				fmt.Println(err)
				continue
			}
			go t.handleConn(conn)
		}
	}
}
func (t *TCPTransport) Stop() {
	t.stopChan <- struct{}{}
}
func (t *TCPTransport) Consume() <-chan any {
	return t.outputChan
}
func (t *TCPTransport) handleConn(conn net.Conn) error {
	peer, err := t.HandshakeFn(conn, t.Address, t.Id)
	if err != nil {
		conn.Close()
		return err
	}
	peer.isOutbound = false
	if !t.registerPeer(peer) {
		// drop connection duplicate
		conn.Close()
		return nil
	}
	return t.readLoop(peer)
}
func (t *TCPTransport) readLoop(peer *Peer) error {
	defer peer.conn.Close()
	for {
		msg := TCPMessage{}
		if err := peer.decoder.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Printf("error: %s\n", err)
			continue
		}
		t.outputChan <- msg.Payload
	}
	return nil
}
func (t *TCPTransport) Send(to string, payload any) error {
	peer, exists := t.peers[to]
	if !exists {
		return fmt.Errorf("%w: %s", ErrPeerNotFound, to)
	}
	msg := &TCPMessage{
		From:    t.Address,
		To:      to,
		Payload: payload,
	}
	return peer.encoder.Encode(msg)
}

func (t *TCPTransport) Broadcast(payload any) error {
	for _, peer := range t.peers {
		if err := t.Send(peer.address, payload); err != nil {
			log.Printf("[TCP]: Err broadcasting msg to %s\n", peer.address)
		}
	}
	return nil
}
