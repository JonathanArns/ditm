package api

import (
	"encoding/json"
	"log"
	"net/http"
)

func (api ApiServer) HandlerDeposit(w http.ResponseWriter, r *http.Request) {
	var force bool
	_force := r.URL.Query().Get("force")
	if _force == "true" {
		force = true
	}
	var req DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[handler:Deposit] could not decode r.Body: %v\n", err)
		http.Error(w, "could not decode r.Body", http.StatusBadRequest)
		return
	}

	status, err := api.commandSvc.Deposit(r.Context(), req.ID, req.Amount, force)
	if err != nil {
		log.Printf("[handler:Deposit] could not deposit to account (perform re-try): %v\n", err)
		status, err = api.commandSvc.Deposit(r.Context(), req.ID, req.Amount, force)
		if err != nil {
			log.Printf("[handler:Deposit:re-try] could not deposit to account (final): %v\n", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if err := json.NewEncoder(w).Encode(DepositResponse{Status: status}); err != nil {
		log.Printf("[handler:Deposit] could not encode r.Body: %v\n", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
