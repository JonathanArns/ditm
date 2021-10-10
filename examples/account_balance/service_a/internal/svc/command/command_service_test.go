package command

import (
	"context"
	"testing"

	"github.com/JonathanArns/ditm/examples/account_balance/service_a/internal/domain"
	"github.com/JonathanArns/ditm/examples/account_balance/service_a/internal/ports"
	"github.com/JonathanArns/ditm/examples/account_balance/service_a/pkg/inmemory"
)

func TestDeposit(t *testing.T) {
	// t.Parallel()

	repo := inmemory.New()
	svc := New(repo)

	testAccount := domain.Account{
		ID:           0,
		AccountOwner: "test",
		Balance:      10.00,
	}

	fillMockDB(t, repo, testAccount)

	tt := []struct {
		name        string
		testAccount int
		amount      float64
		want        float64
		iserr       bool
	}{
		{
			name:        "allowed deposit transaction",
			testAccount: 0,
			amount:      10.00,
			want:        testAccount.Balance + 10.00,
			iserr:       false,
		},
		{
			name:        "negative deposit transaction",
			testAccount: 0,
			amount:      -10.00,
			want:        testAccount.Balance,
			iserr:       true,
		},
	}

	ctx := context.Background()
	for _, tc := range tt {
		err := svc.Deposit(ctx, tc.testAccount, tc.amount)
		if tc.iserr && err == nil || !tc.iserr && err != nil {
			t.Fatalf("[%s] want-err: %v, got-err: %v", tc.name, tc.iserr, err)
		}

		// only check for balance if no error is expected by the
		// test case
		if !tc.iserr {
			account, err := repo.Acquire(ctx, tc.testAccount)
			if err != nil {
				t.Fatalf("[%s] could not load account: %v", tc.name, err)
			}
			if account.Current() != tc.want {
				t.Fatalf("[%s] want-balance: %v, got-balance: %v", tc.name, tc.want, account.Current())
			}
			repo.Release(ctx, account.ID)
		}
	}
}

func TestWithdraw(t *testing.T) {
	t.Parallel()

	repo := inmemory.New()
	svc := New(repo)

	testAccount := domain.Account{
		ID:           0,
		AccountOwner: "test",
		Balance:      10.00,
	}

	fillMockDB(t, repo, testAccount)

	tt := []struct {
		name        string
		testAccount int
		amount      float64
		want        float64
		iserr       bool
	}{
		{
			name:        "allowed withdraw transaction",
			testAccount: 0,
			amount:      10.00,
			want:        (testAccount.Balance - 10.00),
			iserr:       false,
		},
		{
			name:        "negative withdraw transaction",
			testAccount: 0,
			amount:      -10.00,
			want:        testAccount.Balance,
			iserr:       true,
		},
	}

	ctx := context.Background()
	for _, tc := range tt {
		err := svc.Withdraw(ctx, tc.testAccount, tc.amount)
		if tc.iserr && err == nil || !tc.iserr && err != nil {
			t.Fatalf("[%s] want-err: %v, got-err: %v", tc.name, tc.iserr, err)
		}

		// only check for balance if no error is expected by the
		// test case
		if !tc.iserr {
			account, err := repo.Acquire(ctx, tc.testAccount)
			if err != nil {
				t.Fatalf("[%s] could not load account: %v", tc.name, err)
			}

			if account.Current() != tc.want {
				t.Fatalf("[%s] want-balance: %v, got-balance: %v", tc.name, tc.want, account.Current())
			}
			repo.Release(ctx, account.ID)
		}
	}
}

func fillMockDB(t *testing.T, repo ports.Repository, accs ...domain.Account) {
	ctx := context.Background()
	for _, account := range accs {
		if err := repo.Save(ctx, account); err != nil {
			t.Fatalf("could not pre-fill repository: %v", err)
		}
	}
}
