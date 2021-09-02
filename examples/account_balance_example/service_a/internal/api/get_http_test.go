package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"testing"

	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/internal/domain"
	"github.com/gorilla/mux"
)

func TestGetAccount(t *testing.T) {
	t.Parallel()
	apiServer := mockAPI(mux.NewRouter(), domain.Account{ID: 0, AccountOwner: "Test1"})

	tt := []struct {
		name      string
		accountID int
		status    int
		account   domain.Account
	}{
		{
			name:      "get existing account",
			accountID: 0,
			status:    http.StatusOK,
			account:   domain.Account{ID: 0, AccountOwner: "Test1"},
		},
	}

	for _, tc := range tt {
		recorder := httptest.NewRecorder()
		req, err := http.NewRequest("GET", fmt.Sprintf("/api/account/%s", strconv.Itoa(tc.accountID)), nil)
		if err != nil {
			t.Fatalf("[%s] could not create request: %v", tc.name, err)
		}
		req.RequestURI = fmt.Sprintf("/api/account/%d", tc.accountID)

		req = setMuxVars(req, map[string]string{"id": strconv.Itoa(tc.accountID)})

		apiServer.HandlerGet(recorder, req)

		if recorder.Result().StatusCode != tc.status {
			t.Fatalf("[%s] want-status: %v, got-status: %v", tc.name, tc.status, recorder.Result().StatusCode)
		}

		var getRespo GetResponse
		if err := json.NewDecoder(recorder.Body).Decode(&getRespo); err != nil {
			t.Fatalf("[%s] could not decode response.Body: %v", tc.name, err)
		}
		if !reflect.DeepEqual(tc.account, getRespo.Account) {
			t.Fatalf("[%s] returned account differs from the expected one: want: %v, got: %v", tc.name, tc.account, getRespo.Account)
		}
	}
}

func setMuxVars(r *http.Request, vars map[string]string) *http.Request {
	return mux.SetURLVars(r, vars)
}
