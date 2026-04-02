package transport

import (
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net"
)

var (
	ErrPeerNotFound = errors.New("peer not found")
)

type Transport interface {
	Start() error
	Stop()
	Send(string, []byte) error
	Broadcast([]byte) error
	Consume() <-chan []byte
}
type TCPMessage struct {
	Payload []byte
}
type Peer struct {
	conn    net.Conn
	decoder *gob.Decoder
	encoder *gob.Encoder
	address string
}

type TCPTransport struct {
	Address    string
	ln         net.Listener
	stopChan   chan struct{}
	outputChan chan []byte
	peers      map[string]*Peer
}

func New(address string) *TCPTransport {
	return &TCPTransport{
		Address:    address,
		stopChan:   make(chan struct{}),
		outputChan: make(chan []byte, 10),
		peers:      make(map[string]*Peer),
	}
}

func (t *TCPTransport) Start() error {
	ln, err := net.Listen("tcp", t.Address)
	if err != nil {
		return err
	}
	t.ln = ln
	log.Printf("[TCP]: Listening on port: %s...\n", t.Address)
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
func (t *TCPTransport) Consume() <-chan []byte {
	return t.outputChan
}
func (t *TCPTransport) handleConn(conn net.Conn) {
	// TODO: handshake
	peer := &Peer{
		conn:    conn,
		decoder: gob.NewDecoder(conn),
		encoder: gob.NewEncoder(conn),
	}
	for {
		msg := TCPMessage{}
		if err := peer.decoder.Decode(&msg); err != nil {
			fmt.Printf("error: %s\n", err)
		}
		fmt.Printf("Received msg: %+v\n", msg)
	}
}

func (t *TCPTransport) Send(to string, payload []byte) error {
	peer, exists := t.peers[to]
	if !exists {
		return fmt.Errorf("%w: %s", ErrPeerNotFound, to)
	}
	msg := &TCPMessage{
		Payload: payload,
	}
	return peer.encoder.Encode(msg)
}

func (t *TCPTransport) Broadcast(payload []byte) error {
	for _, peer := range t.peers {
		if err := t.Send(peer.address, payload); err != nil {
			log.Printf("[TCP]: Err broadcasting msg to %s\n", peer.address)
		}
	}
	return nil
}
