package sorder

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"github.com/m1ll3r1337/order-service/internal/app/entity"
	"github.com/m1ll3r1337/order-service/internal/app/repository"
	"github.com/m1ll3r1337/order-service/internal/app/service"
)

type srv struct {
	repoOrder repository.Order
}

func NewService(repoOrder repository.Order) service.Order {
	return &srv{repoOrder: repoOrder}
}

func (s *srv) Create(ctx context.Context, req entity.RequestOrderCreate) (entity.Order, error) {
	now := time.Now()

	orderGUID := uuid.Must(uuid.NewV4())

	var totalPrice int64

	items := make([]entity.OrderItem, 0, len(req.Items))
	for _, item := range req.Items {
		totalPrice += int64(item.Quantity) * item.UnitPrice

		items = append(items, entity.OrderItem{
			GUID:        uuid.Must(uuid.NewV4()),
			OrderGUID:   orderGUID,
			ProductGUID: item.ProductGUID,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
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

	err := s.repoOrder.Create(ctx, order)
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
