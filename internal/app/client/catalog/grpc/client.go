package cgrpc

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	ccatalog "github.com/m1ll3r1337/order-service/internal/app/client/catalog"
	"github.com/m1ll3r1337/order-service/internal/app/entity"
	catalogv1 "github.com/m1ll3r1337/order-service/internal/pkg/grpc/gen/catalog/v1"
)

type client struct {
	raw catalogv1.CatalogServiceClient
}

var _ ccatalog.Client = (*client)(nil)

func NewClient(address string) (ccatalog.Client, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}

	raw := catalogv1.NewCatalogServiceClient(conn)
	client := client{
		raw: raw,
	}

	return &client, conn, err
}

func (c *client) Ping(ctx context.Context) error {
	pCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := c.raw.GetProducts(pCtx, &catalogv1.GetProductsRequest{
		Guids: []string{uuid.Nil.String()},
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *client) GetProduct(ctx context.Context, req *catalogv1.GetProductRequest) (*catalogv1.GetProductResponse, error) {
	r, err := c.raw.GetProduct(ctx, req)
	if err != nil {
		return nil, normalizeError(err)
	}

	return r, nil
}

func (c *client) GetProducts(ctx context.Context, req *catalogv1.GetProductsRequest) (*catalogv1.GetProductsResponse, error) {
	r, err := c.raw.GetProducts(ctx, req)
	if err != nil {
		return nil, normalizeError(err)
	}

	return r, nil
}

var grpcCodeToAppError = map[codes.Code]error{
	codes.NotFound:        entity.ErrNotFound,
	codes.InvalidArgument: entity.ErrIncorrectParameters,
	codes.AlreadyExists:   entity.ErrAlreadyExists,
}

func normalizeError(err error) error {
	s, ok := status.FromError(err)
	if !ok {
		return entity.ErrInternal
	}

	mapped, ok := grpcCodeToAppError[s.Code()]
	if !ok {
		return entity.ErrInternal
	}

	return mapped
}
