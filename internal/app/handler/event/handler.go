package ehandler

import (
	"context"

	"github.com/m1ll3r1337/order-service/internal/app/entity"
	"github.com/m1ll3r1337/order-service/internal/pkg/broker"
)

type (
	DeliveryCalculated interface {
		CallbackDeliveryCalculated(
			ctx context.Context,
			ev *entity.EventOrderDeliveryCalculated, headers []broker.Header,
		) error
	}
)
