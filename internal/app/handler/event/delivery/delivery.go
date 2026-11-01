package edelivery

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"

	"github.com/m1ll3r1337/order-service/internal/app/entity"
	ehandler "github.com/m1ll3r1337/order-service/internal/app/handler/event"
	"github.com/m1ll3r1337/order-service/internal/app/service"
	"github.com/m1ll3r1337/order-service/internal/pkg/broker"
)

type handler struct {
	service service.Delivery
}

func NewHandler(service service.Delivery) ehandler.DeliveryCalculated {
	return &handler{
		service: service,
	}
}

func (h *handler) CallbackDeliveryCalculated(
	ctx context.Context,
	ev *entity.EventOrderDeliveryCalculated, headers []broker.Header,
) error {
	log.Info().Ctx(ctx).EmbedObject(ev).Msg("Получено событие order.delivery.calculated")

	if ev.OrderGUID == "" {
		return broker.NotCriticalError(fmt.Errorf("order.delivery.calculated: empty order_guid"))
	}

	guid, err := uuid.FromString(ev.OrderGUID)
	if err != nil {
		return broker.NotCriticalError(fmt.Errorf("order.delivery.calculated: invalid order_guid"))
	}

	applied, err := h.service.ApplyDeliveryCalculated(ctx, guid, ev.DeliveryPrice)
	if err != nil {
		return fmt.Errorf("apply delivery calculated for %s: %w", guid, err)
	}
	if !applied {
		log.Warn().Ctx(ctx).Str("order_guid", ev.OrderGUID).
			Msg("order.delivery.calculated skipped: order is not pending or not found")
		return nil
	}

	log.Info().Ctx(ctx).EmbedObject(ev).Msg("delivery calculated applied")

	return nil
}
