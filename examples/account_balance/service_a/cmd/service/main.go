package main

import (
	"log"

	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/internal/api"
	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/internal/svc/command"
	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/internal/svc/query"
	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/pkg/inmemory"
	"github.com/gorilla/mux"
)

func main() {

	repo := inmemory.New()

	querySvc := query.New(repo)
	commandSvc := command.New(repo)

	apiServer := api.New(mux.NewRouter(), querySvc, commandSvc)

	log.Fatal(apiServer.Serve(":8080"))
}
