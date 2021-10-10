package api

import "github.com/JonathanArns/ditm/examples/account_balance/service_a/internal/domain"

type CreateRequest struct {
	ID           int    `json:"ID"`
	AccountOwner string `json:"account_owner"`
}

type CreateResponse struct {
	Status int `json:"status"`
}

type GetResponse struct {
	Status  int            `json:"status"`
	Account domain.Account `json:"account"`
}

type DepositRequest struct {
	ID     int     `json:"ID"`
	Amount float64 `json:"amount"`
	// Force indicated whether the response
	// should not be send, returning NOTHING
	Force bool `json:"force"`
}

type DepositResponse struct {
	Status int `json:"status"`
}

type WithdrawRequest struct {
	ID     int     `json:"ID"`
	Amount float64 `json:"amount"`
}

type WithdrawResponse struct {
	Status int `json:"status"`
}
