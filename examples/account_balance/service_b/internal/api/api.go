package api

import (
	"net/http"

	middelware "github.com/KonstantinGasser/ditm/examples/account_balance_example/service_b/internal/api/middleware"
	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_b/internal/ports"
	"github.com/gorilla/mux"
)

type ApiServer struct {
	router     *mux.Router
	querySvc   ports.QuerySvc
	commandSvc ports.CommandSvc
}

func (api ApiServer) Serve(addr string) error {
	return http.ListenAndServe(addr, api.router)
}

func New(router *mux.Router, querySvc ports.QuerySvc, commandSvc ports.CommandSvc) *ApiServer {
	api := ApiServer{
		router:     router,
		querySvc:   querySvc,
		commandSvc: commandSvc,
	}
	api.setUp()
	return &api
}

func (api ApiServer) setUp() {
	api.router.HandleFunc("/api/account", api.HandlerCreateAccount).Methods("PUT")
	api.router.HandleFunc("/api/account/{id:[0-9]+}", api.HandlerGet).Methods("GET")
	api.router.HandleFunc("/api/account/deposit", api.HandlerDeposit).Methods("POST")
	api.router.HandleFunc("/api/account/withdraw", nil).Methods("POST")

	api.router.Use(
		middelware.WithLogging,
		middelware.WithIP,
	)
}
