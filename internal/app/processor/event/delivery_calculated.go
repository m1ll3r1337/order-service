package eprocessor

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"

	"github.com/m1ll3r1337/order-service/internal/app/entity"
	ehandler "github.com/m1ll3r1337/order-service/internal/app/handler/event"
	"github.com/m1ll3r1337/order-service/internal/app/processor"
	"github.com/m1ll3r1337/order-service/internal/pkg/broker"
)

type deliveryCalculatedProc struct {
	h   ehandler.DeliveryCalculated
	bus broker.Bus[entity.EventOrderDeliveryCalculated]
}

func NewDeliveryCalculatedEventCatcher(
	h ehandler.DeliveryCalculated,
	bus broker.Bus[entity.EventOrderDeliveryCalculated],
) processor.Processor {
	return &deliveryCalculatedProc{
		h:   h,
		bus: bus,
	}
}

func (p *deliveryCalculatedProc) StartAsync(ctx context.Context, wg *sync.WaitGroup) {
	err := p.bus.Subscribe(ctx, wg, p.h.CallbackDeliveryCalculated)
	if err != nil {
		log.Fatal().Err(err).Str("topic_name", p.bus.QueueName()).Msg("failed to subscribe")
	}

	log.Info().Str("topic_name", p.bus.QueueName()).Msg("subscribed successfully")
}
