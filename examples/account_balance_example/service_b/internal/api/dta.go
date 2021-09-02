package api

type CreateRequest struct {
	ID           int    `json:"ID"`
	AccountOwner string `json:"account_owner"`
}

type CreateResponse struct {
	Status int `json:"status"`
}

type DepositRequest struct {
	ID     int     `json:"ID"`
	Amount float64 `json:"amount"`
}

type DepositResponse struct {
	Status int `json:"status"`
}
