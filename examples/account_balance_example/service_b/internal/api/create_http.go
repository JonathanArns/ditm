package api

import (
	"encoding/json"
	"log"
	"net/http"
)

func (api ApiServer) HandlerCreateAccount(w http.ResponseWriter, r *http.Request) {

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[handler:CreateAccount] could not decode r.Body: %v\n", err)
		http.Error(w, "could not decode r.Body", http.StatusBadRequest)
		return
	}
	status, err := api.commandSvc.CreateAccount(r.Context(), req.ID, req.AccountOwner)
	if err != nil {
		log.Printf("[handler:CreateAccount] could not create account: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if err := json.NewEncoder(w).Encode(CreateResponse{Status: status}); err != nil {
		log.Printf("[handler:CreateAccount] could not encode r.Body: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
