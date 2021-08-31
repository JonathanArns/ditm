package command

import (
	"context"
	"net/http"

	"github.com/KonstantinGasser/ditm/examples/service_b/internal/ports"
)

type service struct {
	cmdClient ports.ServiceAClient
}

func New(client ports.ServiceAClient) ports.CommandSvc {
	return &service{
		cmdClient: client,
	}
}

func (svc service) CreateAccount(ctx context.Context, ID int, accountOwner string) (int, error) {
	status, err := svc.cmdClient.CreateAccount(ctx, ID, accountOwner)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return status, nil
}

func (svc service) Deposit(ctx context.Context, ID int, amount float64, force bool) (int, error) {
	status, err := svc.cmdClient.Deposit(ctx, ID, amount, force)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return status, nil
}

func (svc service) Withdraw(ctx context.Context, ID int, amount float64) (int, error) {
	status, err := svc.cmdClient.Withdraw(ctx, ID, amount)
	if err != nil {
		return http.StatusInternalServerError, err
	}
	return status, nil
}
