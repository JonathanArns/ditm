package ports

import (
	"context"

	"github.com/JonathanArns/ditm/examples/account_balance/service_a/internal/domain"
)

type QuerySvc interface {
	GetAccount(ctx context.Context, ID int) (domain.Account, error)
}
