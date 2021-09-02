package query

import (
	"context"

	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/internal/domain"
	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/internal/ports"
)

type service struct {
	repo ports.Repository
}

func New(repo ports.Repository) ports.QuerySvc {
	return &service{
		repo: repo,
	}
}

func (svc service) GetAccount(ctx context.Context, ID int) (domain.Account, error) {
	defer svc.repo.Release(ctx, ID)
	return svc.repo.Acquire(ctx, ID)
}
