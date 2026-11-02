package sdelivery

import (
	"context"

	"github.com/gofrs/uuid"

	"github.com/m1ll3r1337/order-service/internal/app/repository"
	"github.com/m1ll3r1337/order-service/internal/app/service"
)

type srv struct {
	repoOrder repository.Order
}

func NewService(repoOrder repository.Order) service.Delivery {
	return &srv{
		repoOrder: repoOrder,
	}
}

func (s *srv) ApplyDeliveryCalculated(ctx context.Context, guid uuid.UUID, deliveryPrice int64) (bool, error) {
	return s.repoOrder.ApplyDeliveryCalculated(ctx, guid, deliveryPrice)
}
