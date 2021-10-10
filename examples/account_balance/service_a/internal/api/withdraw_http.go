package api

import (
	"encoding/json"
	"net/http"

	"github.com/JonathanArns/ditm/examples/account_balance/service_a/internal/domain"
	"github.com/JonathanArns/ditm/examples/account_balance/service_a/pkg/inmemory"
)

func (api ApiServer) HandlerWithdraw(w http.ResponseWriter, r *http.Request) {

	var req WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "could not r.Body", http.StatusBadRequest)
		return
	}

	err := api.commandSvc.Withdraw(r.Context(), req.ID, req.Amount)
	if err != nil {
		if err == domain.ErrBalanceThreshold {
			http.Error(w, "balance cannot go negative", http.StatusBadRequest)
			return
		}
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
	if err := json.NewEncoder(w).Encode(WithdrawResponse{
		Status: http.StatusOK,
	}); err != nil {
		http.Error(w, "could not decode response", http.StatusInternalServerError)
		return
	}
}
