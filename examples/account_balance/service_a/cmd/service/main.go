package main

import (
	"log"

	"github.com/JonathanArns/ditm/examples/account_balance/service_a/internal/api"
	"github.com/JonathanArns/ditm/examples/account_balance/service_a/internal/svc/command"
	"github.com/JonathanArns/ditm/examples/account_balance/service_a/internal/svc/query"
	"github.com/JonathanArns/ditm/examples/account_balance/service_a/pkg/inmemory"
	"github.com/gorilla/mux"
)

func main() {

	repo := inmemory.New()

	querySvc := query.New(repo)
	commandSvc := command.New(repo)

	apiServer := api.New(mux.NewRouter(), querySvc, commandSvc)

	log.Fatal(apiServer.Serve(":8080"))
}
