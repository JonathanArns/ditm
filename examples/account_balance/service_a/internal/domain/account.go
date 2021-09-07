package domain

import (
	"fmt"
)

var (
	ErrBalanceThreshold = fmt.Errorf("account's balance threshold reached")
	ErrNegativeAmount   = fmt.Errorf("deposit/withdrawal amount cannot be negative")
)

type Account struct {
	ID           int     `json:"ID"`
	AccountOwner string  `json:"account_owner"`
	Balance      float64 `json:"balance"` // don't do this for real! money must not be represented as floatXX!!
}

func New(ID int, owner string) Account {
	return Account{
		ID:           ID,
		AccountOwner: owner,
		Balance:      0.00,
	}
}

func (a *Account) Withdraw(amount float64) error {
	if amount < 0.00 {
		return ErrNegativeAmount
	}
	// an account's balance should never be less
	// then zero
	if a.Balance == 0.00 {
		return ErrBalanceThreshold
	}
	a.Balance -= amount
	return nil
}

func (a *Account) Deposit(amount float64) error {
	if amount < 0 {
		return ErrNegativeAmount
	}
	a.Balance += amount
	return nil
}

// Current returns the current account's balance
func (a *Account) Current() float64 {
	return a.Balance
}
