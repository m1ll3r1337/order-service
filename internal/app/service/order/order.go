package sorder

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/rs/zerolog/log"

	ccatalog "github.com/m1ll3r1337/order-service/internal/app/client/catalog"
	"github.com/m1ll3r1337/order-service/internal/app/entity"
	"github.com/m1ll3r1337/order-service/internal/app/monitor"
	"github.com/m1ll3r1337/order-service/internal/app/repository"
	"github.com/m1ll3r1337/order-service/internal/app/service"
	"github.com/m1ll3r1337/order-service/internal/pkg/broker"
	catalogv1 "github.com/m1ll3r1337/order-service/internal/pkg/grpc/gen/catalog/v1"
)

type srv struct {
	repoOrder       repository.Order
	catalogGrpc     ccatalog.Client
	metrics         monitor.OrderMetrics
	busOrderCreated broker.Bus[entity.EventOrderCreated]
}

func NewService(
	repoOrder repository.Order,
	catalogGrpc ccatalog.Client,
	metrics monitor.OrderMetrics,
	busOrderCreated broker.Bus[entity.EventOrderCreated],
) service.Order {
	return &srv{
		repoOrder:       repoOrder,
		catalogGrpc:     catalogGrpc,
		metrics:         metrics,
		busOrderCreated: busOrderCreated,
	}
}

func (s *srv) Create(ctx context.Context, req entity.RequestOrderCreate) (entity.Order, error) {
	createMetric := s.metrics.Create()

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
		createMetric.Failed(err)
		return entity.Order{}, err
	}

	prices := make(map[string]int64)
	for _, p := range r.Products {
		prices[p.Guid] = p.Price
	}

	if len(r.MissingGuids) != 0 {
		createMetric.Failed(entity.ErrIncorrectParameters)
		return entity.Order{}, entity.ErrIncorrectParameters
	}

	var totalPrice int64
	items := make([]entity.OrderItem, 0, len(req.Items))
	for _, i := range req.Items {
		price, ok := prices[i.ProductGUID.String()]
		if !ok {
			createMetric.Failed(entity.ErrIncorrectParameters)
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
		createMetric.Failed(err)
		return entity.Order{}, err
	}

	var userGUID *string
	if order.UserGUID != nil {
		guid := order.UserGUID.String()
		userGUID = &guid
	}

	eventOrderCreatedItems := make([]entity.EventOrderCreatedItem, len(order.Items))
	for i, item := range order.Items {
		eventOrderCreatedItems[i] = entity.EventOrderCreatedItem{
			ProductGUID: item.ProductGUID.String(),
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
		}
	}

	ev := entity.EventOrderCreated{
		OrderGUID:  order.GUID.String(),
		UserGUID:   userGUID,
		Currency:   order.Currency,
		TotalPrice: order.TotalPrice,
		Items:      eventOrderCreatedItems,
		CreatedAt:  order.CreatedAt.UTC().Format(time.RFC3339),
	}

	err = s.busOrderCreated.Send(ctx, &ev, entity.BrokerHeaderOrderCreatedType(), entity.BrokerHeaderOrderCreatedEventID())
	if err != nil {
		log.Error().EmbedObject(ev).Err(err).Msg("failed to publish order.created")
		createMetric.PublishFailed()
	}

	createMetric.Success(order.TotalPrice)
	s.metrics.Status().Inc(order.Status)
	return order, nil
}

func (s *srv) GetByGUID(ctx context.Context, guid uuid.UUID) (entity.Order, error) {
	return s.repoOrder.GetByGUID(ctx, guid)
}

func (s *srv) Update(ctx context.Context, guid uuid.UUID, req entity.RequestOrderUpdate) (entity.Order, error) {
	var order entity.Order

	oldStatus := ""
	err := s.repoOrder.InsideTx(ctx, func(txCtx context.Context) error {
		var err error
		order, err = s.repoOrder.GetByGUID(txCtx, guid)
		if err != nil {
			return err
		}

		oldStatus = order.Status
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

	s.metrics.Status().Transition(oldStatus, order.Status)
	return order, nil
}

func (s *srv) Delete(ctx context.Context, guid uuid.UUID) error {
	deletedStatus := ""
	err := s.repoOrder.InsideTx(ctx, func(txCtx context.Context) error {
		order, err := s.repoOrder.GetByGUID(txCtx, guid)
		if err != nil {
			return err
		}

		deletedStatus = order.Status
		err = s.repoOrder.Delete(txCtx, order.GUID)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	s.metrics.Status().Dec(deletedStatus)
	return nil
}

func (s *srv) List(ctx context.Context, req entity.RequestOrderList) ([]entity.Order, error) {
	return s.repoOrder.List(ctx, req.Status, req.UserGUID)
}

//nolint:unused
type noopOrderMetrics struct{}

//nolint:unused
type noopOrderCreateMetric struct{}

//nolint:unused
type noopOrderStatusMetric struct{}

//nolint:unused
func (noopOrderMetrics) Create() monitor.OrderCreateMetric { return noopOrderCreateMetric{} }

//nolint:unused
func (noopOrderMetrics) Status() monitor.OrderStatusMetric { return noopOrderStatusMetric{} }

//nolint:unused
func (noopOrderCreateMetric) Success(int64) {}

//nolint:unused
func (noopOrderCreateMetric) Failed(error) {}

//nolint:unused
func (noopOrderCreateMetric) PublishFailed() {}

//nolint:unused
func (noopOrderStatusMetric) Inc(string) {}

//nolint:unused
func (noopOrderStatusMetric) Dec(string) {}

//nolint:unused
func (noopOrderStatusMetric) Transition(string, string) {}

//nolint:unused
func (noopOrderStatusMetric) Set(string, float64) {}
