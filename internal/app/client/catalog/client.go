package ccatalog

import (
	"context"

	catalogv1 "github.com/m1ll3r1337/order-service/internal/pkg/grpc/gen/catalog/v1"
)

type Client interface {
	Ping(context.Context) error
	GetProduct(context.Context, *catalogv1.GetProductRequest) (*catalogv1.GetProductResponse, error)
	GetProducts(context.Context, *catalogv1.GetProductsRequest) (*catalogv1.GetProductsResponse, error)
}
