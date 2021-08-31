package query

import (
	"context"
	"net/http"

	"github.com/KonstantinGasser/ditm/examples/service_b/internal/ports"
)

type service struct {
	cmdClient ports.ServiceAClient
}

func New(client ports.ServiceAClient) ports.QuerySvc {
	return &service{
		cmdClient: client,
	}
}

func (svc service) GetAccount(ctx context.Context, ID int) (map[string]interface{}, int, error) {
	data, status, err := svc.cmdClient.GetAccount(ctx, ID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	return data, status, nil
}
