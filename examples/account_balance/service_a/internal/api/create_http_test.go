package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/JonathanArns/ditm/examples/account_balance/service_a/internal/domain"
	"github.com/JonathanArns/ditm/examples/account_balance/service_a/internal/svc/command"
	"github.com/JonathanArns/ditm/examples/account_balance/service_a/internal/svc/query"
	"github.com/JonathanArns/ditm/examples/account_balance/service_a/pkg/inmemory"
	"github.com/gorilla/mux"
)

func TestCreateAccount(t *testing.T) {
	t.Parallel()
	apiServer := mockAPI(mux.NewRouter())

	tt := []struct {
		name   string
		body   io.Reader
		status int
	}{
		{
			name:   "create account (OK)",
			body:   bytes.NewReader([]byte(`{"ID": 0, "account_owner": "Test1"}`)),
			status: http.StatusCreated,
		},
		{
			name:   "create account (!OK-duplicate)",
			body:   bytes.NewReader([]byte(`{"ID": 0, "account_owner": "Test1"}`)),
			status: http.StatusBadRequest,
		},
		{
			name:   "create account (invalid-json)",
			body:   bytes.NewReader([]byte(`{"_id": 1, "account": "Test1"}`)),
			status: http.StatusBadRequest,
		},
	}

	for _, tc := range tt {
		req, err := http.NewRequest("PUT", "/api/account", tc.body)
		if err != nil {
			t.Fatalf("[%s] could not create request: %v", tc.name, err)
		}
		recorder := httptest.NewRecorder()
		apiServer.HandleCreateAccount(recorder, req)

		if recorder.Result().StatusCode != tc.status {
			t.Fatalf("[%s] want-status: %v, got-status: %v", tc.name, tc.status, recorder.Result().StatusCode)
		}
	}
}

func mockAPI(router *mux.Router, accounts ...domain.Account) *ApiServer {
	ctx := context.Background()

	repo := inmemory.New()
	for _, account := range accounts {
		if err := repo.Save(ctx, account); err != nil {
			panic("could not save account in pre-fill of datbase")
		}
	}
	cmdSvc := command.New(repo)
	querySvc := query.New(repo)
	server := New(router, querySvc, cmdSvc)
	return server
}
