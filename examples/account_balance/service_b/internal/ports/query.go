package ports

import "context"

type QuerySvc interface {
	GetAccount(ctx context.Context, ID int) (map[string]interface{}, int, error)
}
