package ports

import "context"

type ServiceAClient interface {
	CreateAccount(ctx context.Context, ID int, accountOnwer string) (int, error)
	GetAccount(ctx context.Context, ID int) (map[string]interface{}, int, error)
	Deposit(ctx context.Context, ID int, amount float64, force bool) (int, error)
	Withdraw(ctx context.Context, ID int, amount float64) (int, error)
}
