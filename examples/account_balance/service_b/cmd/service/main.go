package main

import (
	"log"

	"github.com/JonathanArns/ditm/examples/account_balance/service_b/internal/api"
	"github.com/JonathanArns/ditm/examples/account_balance/service_b/internal/svc/command"
	"github.com/JonathanArns/ditm/examples/account_balance/service_b/internal/svc/query"
	"github.com/JonathanArns/ditm/examples/account_balance/service_b/pkg/httpclient"
	"github.com/gorilla/mux"
)

func main() {
	client := httpclient.New()

	querySvc := query.New(client)
	commandSvc := command.New(client)

	apiServer := api.New(mux.NewRouter(), querySvc, commandSvc)

	log.Fatal(apiServer.Serve(":80"))
}
