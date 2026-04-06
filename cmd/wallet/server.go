package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/GiorgosMarga/blockchain/wallet"
)

type Server struct {
	wallet     *wallet.Wallet
	listenAddr string
	username   string
	templates  *template.Template
}

type PageData struct {
	Username string
	Balance  uint64
	Contacts []wallet.Recipient
}

func NewServer(listenAddr string, w *wallet.Wallet, username string) *Server {
	tmpl := template.Must(template.ParseFiles(
		"./cmd/wallet/templates/index.html",
		"./cmd/wallet/templates/balance.html",
		"./cmd/wallet/templates/contacts.html",
		"./cmd/wallet/templates/message.html",
	))

	return &Server{
		wallet:     w,
		listenAddr: listenAddr,
		username:   username,
		templates:  tmpl,
	}
}
func (s *Server) Start() error {
	http.HandleFunc("/", s.handleHome)
	http.HandleFunc("/balance", s.handleBalance)
	http.HandleFunc("/update-balance", s.handleUpdateBalance)
	http.HandleFunc("/send", s.handleSend)
	http.HandleFunc("/contacts", s.handleContacts)
	http.HandleFunc("/add-recipient", s.handleAddRecipient)

	log.Printf("Starting wallet server on %s", s.listenAddr)
	return http.ListenAndServe(s.listenAddr, nil)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	data := PageData{
		Username: s.username,
		Balance:  s.wallet.Balance(),
		Contacts: s.wallet.Core.Config.Contacts,
	}
	if err := s.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		fmt.Println(err)
	}
}

func (s *Server) handleBalance(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Balance": s.wallet.Balance(),
	}
	s.templates.ExecuteTemplate(w, "balance.html", data)
}

func (s *Server) handleUpdateBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := s.wallet.UpdateBalance()

	if err != nil {
		data := map[string]interface{}{
			"Message": "Error updating balance: " + err.Error(),
			"Type":    "error",
		}
		s.templates.ExecuteTemplate(w, "message.html", data)
		return
	}

	// Return updated balance
	data := map[string]interface{}{
		"Balance": s.wallet.Balance(),
	}
	s.templates.ExecuteTemplate(w, "balance.html", data)
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	address := r.FormValue("address")
	amountStr := r.FormValue("amount")

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		data := map[string]interface{}{
			"Message": "Invalid amount",
			"Type":    "error",
		}
		s.templates.ExecuteTemplate(w, "message.html", data)
		return
	}

	// TODO: Implement actual send logic using your wallet
	// Example: err := s.wallet.Send(address, amount)

	data := map[string]interface{}{
		"Message": fmt.Sprintf("Sent %d coins to %s", amount, address),
		"Type":    "success",
	}
	s.templates.ExecuteTemplate(w, "message.html", data)
}

func (s *Server) handleContacts(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Contacts": s.wallet.Core.Config.Contacts,
	}
	s.templates.ExecuteTemplate(w, "contacts.html", data)
}

func (s *Server) handleAddRecipient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	name := r.FormValue("name")
	address := r.FormValue("address")

	pubKey, _ := ecdsa.ParseUncompressedPublicKey(elliptic.P256(), []byte(address))

	s.wallet.Core.Config.Contacts = append(s.wallet.Core.Config.Contacts, wallet.Recipient{
		Name:   name,
		PubKey: *pubKey,
	})

	// Return updated Contacts list
	data := map[string]interface{}{
		"Contacts": s.wallet.Core.Config.Contacts,
	}
	s.templates.ExecuteTemplate(w, "Contacts.html", data)
}
