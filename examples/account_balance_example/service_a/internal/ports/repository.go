package ports

import (
	"context"

	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/internal/domain"
)

// Repository holds all functions a storage adapter needs
// to provide
type Repository interface {
	Create(ctx context.Context, account domain.Account) error
	Acquire(ctx context.Context, ID int) (domain.Account, error)
	Release(ctx context.Context, ID int) error
	Save(ctx context.Context, account domain.Account) error
}
