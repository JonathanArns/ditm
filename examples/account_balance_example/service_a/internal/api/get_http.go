package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/pkg/inmemory"
	"github.com/gorilla/mux"
)

// HandlerGet retunrs an stored account
func (api ApiServer) HandlerGet(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	varID, ok := vars["id"]
	if !ok {
		http.Error(w, "could not find account id", http.StatusBadRequest)
		return
	}
	ID, err := strconv.Atoi(varID)
	if err != nil {
		http.Error(w, "account id must be an integer", http.StatusBadRequest)
		return
	}

	account, err := api.querySvc.GetAccount(r.Context(), ID)
	if err != nil {
		if err == inmemory.ErrNoSuchAccount {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, "could not get account details", http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(GetResponse{
		Status:  http.StatusOK,
		Account: account,
	}); err != nil {
		http.Error(w, "could not decode response", http.StatusInternalServerError)
		return
	}
}
