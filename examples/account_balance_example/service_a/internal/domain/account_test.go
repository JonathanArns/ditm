package domain

import (
	"testing"
)

func TestAddBalance(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name    string
		account *Account
		amount  float64
		want    float64
		err     error
	}{
		{
			name:    "add to blanace",
			account: &Account{ID: 0, AccountOwner: "test", Balance: 100.00},
			amount:  50.00,
			want:    float64(100.00 + 50.00),
			err:     nil,
		},
		{
			name:    "invalid transaction (balance 0.00)",
			account: &Account{ID: 2, AccountOwner: "test", Balance: 0.00},
			amount:  -50.00,
			want:    0.00,
			err:     ErrNegativeAmount,
		},
		{
			name:    "add to zero balance",
			account: &Account{ID: 3, AccountOwner: "test", Balance: 0.00},
			amount:  50.00,
			want:    50.00,
			err:     nil,
		},
	}

	for _, tc := range tt {
		err := tc.account.Deposit(tc.amount)
		if err != tc.err {
			t.Fatalf("[%s] want-err: %v, got-err: %v", tc.name, tc.err, err)
		}

		if tc.account.Balance != tc.want {
			t.Fatalf("[%s] want-amount: %v, got-amount: %v", tc.name, tc.want, tc.account.Balance)
		}
	}
}

func TestSubBalance(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name    string
		account *Account
		amount  float64
		want    float64
		err     error
	}{
		{
			name:    "sub to balance",
			account: &Account{ID: 0, AccountOwner: "test", Balance: 100.00},
			amount:  50.00,
			want:    float64(100.00 - 50.00),
			err:     nil,
		},
		{
			name:    "invalid withdraw (negative amount)",
			account: &Account{ID: 2, AccountOwner: "test", Balance: 0.00},
			amount:  -50.00,
			want:    0.00,
			err:     ErrNegativeAmount,
		},
		{
			name:    "withdraw from zero balance",
			account: &Account{ID: 3, AccountOwner: "test", Balance: 0.00},
			amount:  50.00,
			want:    0.00,
			err:     ErrBalanceThreshold,
		},
	}

	for _, tc := range tt {
		err := tc.account.Withdraw(tc.amount)
		if err != tc.err {
			t.Fatalf("[%s] want-err: %v, got-err: %v", tc.name, tc.err, err)
		}

		if tc.account.Balance != tc.want {
			t.Fatalf("[%s] want-amount: %v, got-amount: %v", tc.name, tc.want, tc.account.Balance)
		}
	}
}
