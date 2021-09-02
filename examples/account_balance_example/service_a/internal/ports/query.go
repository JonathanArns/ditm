package ports

import (
	"context"

	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/internal/domain"
)

type QuerySvc interface {
	GetAccount(ctx context.Context, ID int) (domain.Account, error)
}
