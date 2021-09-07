package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/internal/domain"
	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/pkg/inmemory"
)

func (api ApiServer) HandlerDeposit(w http.ResponseWriter, r *http.Request) {

	var req DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[handler:Deposit] could not decode r.Body: %v\n", err)
		http.Error(w, "could not decode request body", http.StatusBadRequest)
		return
	}
	err := api.commandSvc.Deposit(r.Context(), req.ID, req.Amount)
	if err != nil {
		if err == domain.ErrNegativeAmount {
			http.Error(w, "deposit/withdraw amount cannot be negative", http.StatusBadRequest)
			return
		}
		if err == inmemory.ErrNoSuchAccount {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "could not deposit from account", http.StatusInternalServerError)
		return
	}
	// terminate request unexpectedly by hijacking
	// the http response and closing the connection
	if req.Force {
		log.Printf("[handler:Deposit] force unexpected network failure send no response")
		hj, ok := w.(http.Hijacker)
		if !ok {
			log.Printf("[handler:Deposit] could not create hijacker: %v\n", err)
			return // serve normal response
		}
		conn, buf, err := hj.Hijack()
		if err != nil {
			log.Printf("[handler:Deposit] could not hijack connection: %v\n", err)
			return // serve normal response
		}
		_ = buf.Flush()
		_ = conn.Close()
		return
	}
	if err := json.NewEncoder(w).Encode(DepositResponse{
		Status: http.StatusOK,
	}); err != nil {
		http.Error(w, "could not decode response", http.StatusInternalServerError)
		return
	}
}
