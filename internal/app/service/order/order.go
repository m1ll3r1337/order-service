package sorder

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	ccatalog "github.com/m1ll3r1337/order-service/internal/app/client/catalog"
	"github.com/m1ll3r1337/order-service/internal/app/entity"
	"github.com/m1ll3r1337/order-service/internal/app/repository"
	"github.com/m1ll3r1337/order-service/internal/app/service"
	catalogv1 "github.com/m1ll3r1337/order-service/internal/pkg/grpc/gen/catalog/v1"
)

type srv struct {
	repoOrder   repository.Order
	catalogGrpc ccatalog.Client
}

func NewService(repoOrder repository.Order, catalogGrpc ccatalog.Client) service.Order {
	return &srv{repoOrder: repoOrder, catalogGrpc: catalogGrpc}
}

func (s *srv) Create(ctx context.Context, req entity.RequestOrderCreate) (entity.Order, error) {
	now := time.Now()

	orderGUID := uuid.Must(uuid.NewV4())

	guids := make([]string, 0, len(req.Items))
	for _, item := range req.Items {
		guids = append(guids, item.ProductGUID.String())
	}

	r, err := s.catalogGrpc.GetProducts(ctx, &catalogv1.GetProductsRequest{
		Guids: guids,
	})
	if err != nil {
		return entity.Order{}, err
	}

	prices := make(map[string]int64)
	for _, p := range r.Products {
		prices[p.Guid] = p.Price
	}

	if len(r.MissingGuids) != 0 {
		return entity.Order{}, entity.ErrIncorrectParameters
	}

	var totalPrice int64
	items := make([]entity.OrderItem, 0, len(req.Items))
	for _, i := range req.Items {
		price, ok := prices[i.ProductGUID.String()]
		if !ok {
			return entity.Order{}, entity.ErrIncorrectParameters
		}

		totalPrice += price * int64(i.Quantity)
		items = append(items, entity.OrderItem{
			GUID:        uuid.Must(uuid.NewV4()),
			OrderGUID:   orderGUID,
			ProductGUID: i.ProductGUID,
			Quantity:    i.Quantity,
			UnitPrice:   price,
			CreatedAt:   now,
			UpdatedAt:   now,
		})
	}

	order := entity.Order{
		GUID:       orderGUID,
		UserGUID:   req.UserGUID,
		TotalPrice: totalPrice,
		Currency:   req.Currency,
		Status:     "pending",
		CreatedAt:  now,
		UpdatedAt:  now,
		Items:      items,
	}

	err = s.repoOrder.Create(ctx, order)
	if err != nil {
		return entity.Order{}, err
	}
	return order, nil
}

func (s *srv) GetByGUID(ctx context.Context, guid uuid.UUID) (entity.Order, error) {
	return s.repoOrder.GetByGUID(ctx, guid)
}

func (s *srv) Update(ctx context.Context, guid uuid.UUID, req entity.RequestOrderUpdate) (entity.Order, error) {
	var order entity.Order

	err := s.repoOrder.InsideTx(ctx, func(txCtx context.Context) error {
		var err error
		order, err = s.repoOrder.GetByGUID(txCtx, guid)
		if err != nil {
			return err
		}

		order.Status = req.Status
		order.UpdatedAt = time.Now()

		err = s.repoOrder.Update(txCtx, order)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return entity.Order{}, err
	}

	return order, nil
}

func (s *srv) Delete(ctx context.Context, guid uuid.UUID) error {
	return s.repoOrder.InsideTx(ctx, func(txCtx context.Context) error {
		order, err := s.repoOrder.GetByGUID(txCtx, guid)
		if err != nil {
			return err
		}

		err = s.repoOrder.Delete(txCtx, order.GUID)
		if err != nil {
			return err
		}

		return nil
	})
}

func (s *srv) List(ctx context.Context, req entity.RequestOrderList) ([]entity.Order, error) {
	return s.repoOrder.List(ctx, req.Status, req.UserGUID)
}
