package ports

import (
	"context"
)

type CommandSvc interface {
	Create(ctx context.Context, ID int, accountOwner string) error
	Deposit(ctx context.Context, ID int, amount float64) error
	Withdraw(ctx context.Context, ID int, amount float64) error
}
