package main

import (
	"flag"
	"log"
	"strings"

	"github.com/GiorgosMarga/blockchain/node"
)

func main() {

	var (
		peerNodes  []string
		bcPath     string
		listenAddr string
	)
	flag.StringVar(&listenAddr, "listenAddr", ":3000", "Node's TCP listening address.")
	flag.StringVar(&bcPath, "bc_path", "blockchain_3000", "Filepath to load blockchain from.")
	flag.Func("peerNodes", "Known peer nodes seperated with comma ','.", func(s string) error {
		peerNodes = strings.Split(s, ",")
		return nil
	})

	flag.Parse()

	node := node.NewNode(listenAddr, bcPath, peerNodes...)

	go node.Start()

	s := NewServer(":3001", node)
	log.Fatal(s.Start())
}
