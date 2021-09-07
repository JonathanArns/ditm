package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/pkg/inmemory"
)

func (api ApiServer) HandleCreateAccount(w http.ResponseWriter, r *http.Request) {

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[handler:CreateAccount] could not decode r.Body: %v\n", err)
		http.Error(w, "could not decode request body", http.StatusBadRequest)
		return
	}

	err := api.commandSvc.Create(r.Context(), req.ID, req.AccountOwner)
	if err != nil {
		log.Printf("[handler:CreateAccount] %v\n", err)
		if err == inmemory.ErrDuplicatedAccount {
			http.Error(w, "account ID exists", http.StatusBadRequest)
			return
		}
		http.Error(w, "could not create account", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusCreated)
}
