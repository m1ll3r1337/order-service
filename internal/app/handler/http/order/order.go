package horder

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gofrs/uuid"

	"github.com/m1ll3r1337/order-service/internal/app/entity"
	rhandler "github.com/m1ll3r1337/order-service/internal/app/handler/http"
	"github.com/m1ll3r1337/order-service/internal/app/service"
	"github.com/m1ll3r1337/order-service/internal/pkg/http/httph"
)

type handler struct {
	srv service.Order
}

func NewHandler(srv service.Order) rhandler.Order {
	return &handler{srv: srv}
}

func mapItems(items []entity.OrderItem) []entity.ResponseOrderItem {
	itemResp := make([]entity.ResponseOrderItem, len(items))
	for i, item := range items {
		itemResp[i] = entity.ResponseOrderItem{
			GUID:        item.GUID,
			ProductGUID: item.ProductGUID,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
		}
	}

	return itemResp
}

func (h *handler) Create(c *gin.Context) {
	var req entity.RequestOrderCreate
	if err := c.ShouldBindJSON(&req); err != nil {
		httph.HandleError(c.Writer, c.Request, entity.ErrIncorrectParameters)
		return
	}

	order, err := h.srv.Create(c.Request.Context(), req)
	if err != nil {
		httph.HandleError(c.Writer, c.Request, err)
		return
	}

	resp := entity.ResponseOrderCreate{
		GUID:       order.GUID,
		UserGUID:   order.UserGUID,
		TotalPrice: order.TotalPrice,
		Currency:   order.Currency,
		Status:     order.Status,
		CreatedAt:  order.CreatedAt,
		UpdatedAt:  order.UpdatedAt,
		Items:      mapItems(order.Items),
	}

	httph.SendJSON(c.Writer, http.StatusCreated, resp)
}

func (h *handler) GetByGUID(c *gin.Context) {
	id, err := uuid.FromString(c.Param("guid"))
	if err != nil {
		httph.HandleError(c.Writer, c.Request, entity.ErrIncorrectParameters)
		return
	}

	order, err := h.srv.GetByGUID(c.Request.Context(), id)
	if err != nil {
		httph.HandleError(c.Writer, c.Request, err)
		return
	}

	resp := entity.ResponseOrderGet{
		GUID:       order.GUID,
		UserGUID:   order.UserGUID,
		TotalPrice: order.TotalPrice,
		Currency:   order.Currency,
		Status:     order.Status,
		CreatedAt:  order.CreatedAt,
		UpdatedAt:  order.UpdatedAt,
		Items:      mapItems(order.Items),
	}

	httph.SendJSON(c.Writer, http.StatusOK, resp)
}

func (h *handler) Update(c *gin.Context) {
	id, err := uuid.FromString(c.Param("guid"))
	if err != nil {
		httph.HandleError(c.Writer, c.Request, entity.ErrIncorrectParameters)
		return
	}

	var req entity.RequestOrderUpdate
	if err := c.ShouldBindJSON(&req); err != nil {
		httph.HandleError(c.Writer, c.Request, entity.ErrIncorrectParameters)
		return
	}

	order, err := h.srv.Update(c.Request.Context(), id, req)
	if err != nil {
		httph.HandleError(c.Writer, c.Request, err)
		return
	}

	resp := entity.ResponseOrderUpdate{
		GUID:       order.GUID,
		UserGUID:   order.UserGUID,
		TotalPrice: order.TotalPrice,
		Currency:   order.Currency,
		Status:     order.Status,
		CreatedAt:  order.CreatedAt,
		UpdatedAt:  order.UpdatedAt,
		Items:      mapItems(order.Items),
	}

	httph.SendJSON(c.Writer, http.StatusOK, resp)
}

func (h *handler) Delete(c *gin.Context) {
	id, err := uuid.FromString(c.Param("guid"))
	if err != nil {
		httph.HandleError(c.Writer, c.Request, entity.ErrIncorrectParameters)
		return
	}

	err = h.srv.Delete(c.Request.Context(), id)
	if err != nil {
		httph.HandleError(c.Writer, c.Request, err)
		return
	}

	httph.SendEmpty(c.Writer, http.StatusOK)
}

func (h *handler) List(c *gin.Context) {
	var req entity.RequestOrderList
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			httph.HandleError(c.Writer, c.Request, entity.ErrIncorrectParameters)
			return
		}
	}

	orders, err := h.srv.List(c.Request.Context(), req)
	if err != nil {
		httph.HandleError(c.Writer, c.Request, err)
		return
	}

	items := make([]entity.ResponseOrderListItem, len(orders))
	for i, order := range orders {
		items[i] = entity.ResponseOrderListItem{
			GUID:       order.GUID,
			UserGUID:   order.UserGUID,
			TotalPrice: order.TotalPrice,
			Currency:   order.Currency,
			Status:     order.Status,
			CreatedAt:  order.CreatedAt,
			UpdatedAt:  order.UpdatedAt,
		}
	}

	httph.SendJSON(c.Writer, http.StatusOK, entity.ResponseOrderList{Data: items})
}
