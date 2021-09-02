package query

import (
	"context"
	"testing"

	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/internal/domain"
	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/internal/ports"
	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/pkg/inmemory"
)

func TestGetAccount(t *testing.T) {
	repo := inmemory.New()
	svc := New(repo)

	testAccount := domain.Account{
		ID:           0,
		AccountOwner: "test",
		Balance:      10.00,
	}

	fillMockDB(t, repo, testAccount)

	t.Parallel()

	tt := []struct {
		name  string
		id    int
		want  domain.Account
		iserr bool
	}{
		{
			name:  "lookup existing account",
			id:    0,
			want:  testAccount,
			iserr: false,
		},
		{
			name:  "lookup NOT existing account",
			id:    1,
			want:  domain.Account{},
			iserr: true,
		},
	}
	ctx := context.Background()

	for _, tc := range tt {
		got, err := svc.GetAccount(ctx, tc.id)
		if (!tc.iserr && err != nil) || (tc.iserr && err == nil) {
			t.Fatalf("[%s] want-err: %v, got-err: %v", tc.name, tc.iserr, err)
		}

		if got != tc.want {
			t.Fatalf("[%s] want: %v, got: %v", tc.name, tc.want, got)
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
