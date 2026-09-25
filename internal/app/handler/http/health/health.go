package rhealth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	rhandler "github.com/m1ll3r1337/order-service/internal/app/handler/http"
)

type handler struct{}

func NewHandler() rhandler.Health {
	return &handler{}
}

func (h *handler) LastCheck(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}
