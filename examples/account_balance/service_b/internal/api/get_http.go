package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func (api ApiServer) HandlerGet(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	varID, ok := vars["id"]
	if !ok {
		log.Printf("[handler:GetAccount] could not find account id\n")
		http.Error(w, "could not find account id", http.StatusBadRequest)
		return
	}
	ID, err := strconv.Atoi(varID)
	if err != nil {
		log.Printf("[handler:GetAccount] %v\n", err)
		http.Error(w, "account id must be an integer", http.StatusBadRequest)
		return
	}
	data, status, err := api.querySvc.GetAccount(r.Context(), ID)
	if err != nil {
		log.Printf("[handler:GetAccount] %v\n", err)
		http.Error(w, fmt.Errorf("could not get account: %v", err).Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  status,
		"account": data,
	}); err != nil {
		log.Printf("[handler:GetAccount] %v\n", err)
		http.Error(w, fmt.Errorf("could not marshal response: %v", err).Error(), http.StatusInternalServerError)
		return
	}
}
