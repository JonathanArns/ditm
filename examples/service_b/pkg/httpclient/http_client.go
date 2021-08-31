package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const (
	apiBase          = "http://localhost:8080"
	apiCreateAccount = "/api/account"
	apiGetAccount    = "/api/account"
	apiDeposit       = "/api/account/deposit"
	apiWithdraw      = "/api/account/withdraw"
)

type Client struct {
}

func New() *Client {
	return &Client{}
}

func (c Client) CreateAccount(ctx context.Context, ID int, accountOnwer string) (int, error) {
	client := http.Client{}

	b, err := json.Marshal(map[string]interface{}{
		"id":            ID,
		"account_owner": accountOnwer,
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}
	req, err := http.NewRequest("PUT", remote(apiCreateAccount), bytes.NewReader(b))
	if err != nil {
		return http.StatusInternalServerError, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func (c Client) GetAccount(ctx context.Context, ID int) (map[string]interface{}, int, error) {
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/%d", remote(apiGetAccount), ID), nil)
	if err != nil {
		fmt.Println(err)
		return nil, http.StatusInternalServerError, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, http.StatusInternalServerError, err
	}

	return data, resp.StatusCode, nil
}

func (c Client) Deposit(ctx context.Context, ID int, amount float64, force bool) (int, error) {
	client := http.Client{}

	fmt.Println(force)
	b, err := json.Marshal(map[string]interface{}{
		"id":     ID,
		"amount": amount,
		"force":  force,
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}
	req, err := http.NewRequest(http.MethodPost, remote(apiDeposit), bytes.NewReader(b))
	if err != nil {
		return http.StatusInternalServerError, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	fmt.Println(resp)
	fmt.Println(err)
	return resp.StatusCode, nil
}

func (c Client) Withdraw(ctx context.Context, ID int, amount float64) (int, error) {
	client := http.Client{}

	b, err := json.Marshal(map[string]interface{}{
		"id":     ID,
		"amount": amount,
	})
	if err != nil {
		return http.StatusInternalServerError, err
	}
	req, err := http.NewRequest(http.MethodPost, remote(apiWithdraw), bytes.NewReader(b))
	if err != nil {
		return http.StatusInternalServerError, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func remote(apiRoute string) string {
	return fmt.Sprintf("%v%v", apiBase, apiRoute)
}
