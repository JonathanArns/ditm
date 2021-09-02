package ports

import "context"

type CommandSvc interface {
	CreateAccount(ctx context.Context, ID int, accountOwner string) (int, error)
	Deposit(ctx context.Context, ID int, amount float64, force bool) (int, error)
	Withdraw(ctx context.Context, ID int, amount float64) (int, error)
}
