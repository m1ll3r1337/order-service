package entity

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/rs/zerolog"

	"github.com/m1ll3r1337/order-service/internal/pkg/broker"
)

const (
	tableNameOrder     = "orders"
	tableNameOrderItem = "order_items"
)

type Order struct {
	ID            int64      `gorm:"column:id;type:bigint;autoIncrement;unique;"`
	GUID          uuid.UUID  `gorm:"column:guid;type:uuid;primaryKey;"`
	UserGUID      *uuid.UUID `gorm:"column:user_guid;type:uuid;"`
	TotalPrice    int64      `gorm:"column:total_price;type:bigint;"`
	DeliveryPrice int64      `gorm:"column:delivery_price"`
	Currency      string     `gorm:"column:currency;type:varchar(3);"`
	Status        string     `gorm:"column:status;type:varchar(32);"`
	CreatedAt     time.Time  `gorm:"column:created_at;type:timestamptz;"`
	UpdatedAt     time.Time  `gorm:"column:updated_at;type:timestamptz;"`

	Items []OrderItem `gorm:"foreignKey:OrderGUID;references:GUID"`
}

func (Order) TableName() string { return tableNameOrder }

type OrderItem struct {
	ID          int64     `gorm:"column:id;type:bigint;autoIncrement;unique;"`
	GUID        uuid.UUID `gorm:"column:guid;type:uuid;primaryKey;"`
	OrderGUID   uuid.UUID `gorm:"column:order_guid;type:uuid;"`
	ProductGUID uuid.UUID `gorm:"column:product_guid;type:uuid;"`
	Quantity    int       `gorm:"column:quantity;type:integer;"`
	UnitPrice   int64     `gorm:"column:unit_price;type:bigint;"`
	CreatedAt   time.Time `gorm:"column:created_at;type:timestamptz;"`
	UpdatedAt   time.Time `gorm:"column:updated_at;type:timestamptz;"`
}

func (OrderItem) TableName() string { return tableNameOrderItem }

////////////////////////////////////////////////////////////////////////////////
///// EVENT MODEL //////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

type EventOrderCreated struct {
	OrderGUID  string                  `json:"order_guid"`
	UserGUID   *string                 `json:"user_guid,omitempty"`
	Currency   string                  `json:"currency"`
	TotalPrice int64                   `json:"total_price"`
	Items      []EventOrderCreatedItem `json:"items"`
	CreatedAt  string                  `json:"created_at"`
}

type EventOrderCreatedItem struct {
	ProductGUID string `json:"product_guid"`
	Quantity    int    `json:"quantity"`
	UnitPrice   int64  `json:"unit_price"`
}

func (e EventOrderCreated) EventId() string {
	return e.OrderGUID
}

func (e EventOrderCreated) MarshalZerologObject(ev *zerolog.Event) {
	ev.Str("order_guid", e.OrderGUID)
	ev.Str("currency", e.Currency)
	ev.Int64("total_price", e.TotalPrice)
	ev.Int("items_count", len(e.Items))
}

////////////////////////////////////////////////////////////////////////////////
///// EVENT AUXILIARIES ////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

const (
	BrokerHeaderKeyOrderEventType = "type"
	BrokerHeaderKeyOrderEventID   = "event-id"

	BrokerHeaderValueOrderCreated = "order.created"
)

func BrokerHeaderOrderCreatedType() broker.Header {
	return broker.Header{
		Key:   BrokerHeaderKeyOrderEventType,
		Value: BrokerHeaderValueOrderCreated,
	}
}

func BrokerHeaderOrderCreatedEventID() broker.Header {
	return broker.Header{
		Key:   BrokerHeaderKeyOrderEventID,
		Value: uuid.Must(uuid.NewV4()).String(),
	}
}

////////////////////////////////////////////////////////////////////////////////
///// HTTP REQUEST & RESPONSE //////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

type RequestOrderCreate struct {
	UserGUID *uuid.UUID               `json:"user_guid"`
	Currency string                   `json:"currency" binding:"required,len=3"`
	Items    []RequestOrderItemCreate `json:"items"    binding:"required,min=1,dive"`
}

type RequestOrderItemCreate struct {
	ProductGUID uuid.UUID `json:"product_guid" binding:"required"`
	Quantity    int       `json:"quantity"     binding:"required,gt=0"`
}

type RequestOrderUpdate struct {
	Status string `json:"status" binding:"required"`
}

type RequestOrderList struct {
	Status   *string    `json:"status"`
	UserGUID *uuid.UUID `json:"user_guid"`
}

type ResponseOrderCreate struct {
	GUID          uuid.UUID           `json:"guid"`
	UserGUID      *uuid.UUID          `json:"user_guid"`
	TotalPrice    int64               `json:"total_price"`
	DeliveryPrice int64               `json:"delivery_price"`
	Currency      string              `json:"currency"`
	Status        string              `json:"status"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	Items         []ResponseOrderItem `json:"items"`
}

type ResponseOrderGet struct {
	GUID          uuid.UUID           `json:"guid"`
	UserGUID      *uuid.UUID          `json:"user_guid"`
	TotalPrice    int64               `json:"total_price"`
	DeliveryPrice int64               `json:"delivery_price"`
	Currency      string              `json:"currency"`
	Status        string              `json:"status"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	Items         []ResponseOrderItem `json:"items"`
}

type ResponseOrderUpdate struct {
	GUID          uuid.UUID           `json:"guid"`
	UserGUID      *uuid.UUID          `json:"user_guid"`
	TotalPrice    int64               `json:"total_price"`
	DeliveryPrice int64               `json:"delivery_price"`
	Currency      string              `json:"currency"`
	Status        string              `json:"status"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
	Items         []ResponseOrderItem `json:"items"`
}

type ResponseOrderItem struct {
	GUID        uuid.UUID `json:"guid"`
	ProductGUID uuid.UUID `json:"product_guid"`
	Quantity    int       `json:"quantity"`
	UnitPrice   int64     `json:"unit_price"`
}

type ResponseOrderListItem struct {
	GUID       uuid.UUID  `json:"guid"`
	UserGUID   *uuid.UUID `json:"user_guid"`
	TotalPrice int64      `json:"total_price"`
	Currency   string     `json:"currency"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

type ResponseOrderList struct {
	Data []ResponseOrderListItem `json:"data"`
}
