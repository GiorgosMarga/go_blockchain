package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/GiorgosMarga/blockchain/block"
	"github.com/GiorgosMarga/blockchain/node"
)

type Server struct {
	node       *node.Node
	listenAddr string
	templates  *template.Template
}

type PageData struct {
	Blocks     []*block.Block
	BlockCount int
}

func NewServer(listenAddr string, node *node.Node) *Server {
	tmpl := template.Must(template.New("explorer").Funcs(templateFuncs).ParseFiles(
		"./cmd/node/templates/index.html",
		"./cmd/node/templates/blocks.html",
	))
	return &Server{
		node:       node,
		listenAddr: listenAddr,
		templates:  tmpl,
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/", s.handleHome)
	return http.ListenAndServe(s.listenAddr, nil)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Blocks:     s.node.Blockchain.Blocks,
		BlockCount: len(s.node.Blockchain.Blocks),
	}
	if err := s.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Println(err)
	}
}
