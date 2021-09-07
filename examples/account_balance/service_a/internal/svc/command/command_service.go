package command

import (
	"context"

	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/internal/domain"
	"github.com/KonstantinGasser/ditm/examples/account_balance_example/service_a/internal/ports"
)

type service struct {
	repo ports.Repository
}

func New(repo ports.Repository) ports.CommandSvc {
	return &service{
		repo: repo,
	}
}

func (svc service) Create(ctx context.Context, ID int, accountOwner string) error {
	return svc.repo.Create(ctx, domain.New(ID, accountOwner))
}

// Transaction describes the use-case where the balance of an
// account gets modified either by adding or subtracting an amount
// X to/from the account
func (svc service) Deposit(ctx context.Context, ID int, amount float64) error {

	account, err := svc.repo.Acquire(ctx, ID)
	if err != nil {
		return err
	}
	defer svc.repo.Release(ctx, ID)

	if err := account.Deposit(amount); err != nil {
		return err
	}

	if err := svc.repo.Save(ctx, account); err != nil {
		return err
	}
	return nil
}

// Transaction describes the use-case where the balance of an
// account gets modified either by adding or subtracting an amount
// X to/from the account
func (svc service) Withdraw(ctx context.Context, ID int, amount float64) error {

	account, err := svc.repo.Acquire(ctx, ID)
	if err != nil {
		return err
	}
	defer svc.repo.Release(ctx, ID)

	if err := account.Withdraw(amount); err != nil {
		return err
	}

	if err := svc.repo.Save(ctx, account); err != nil {
		return err
	}
	return nil
}
