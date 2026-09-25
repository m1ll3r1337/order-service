package rprocessor

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/m1ll3r1337/order-service/internal/app/config/section"
	rhandler "github.com/m1ll3r1337/order-service/internal/app/handler/http"
)

type httpProc struct {
	server http.Server
	addr   string
}

func NewHTTP(hHealth rhandler.Health, cfg section.ProcessorWebServer) *httpProc {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()
	r.Use(gin.Recovery())
	r.NoRoute(handleNotFound)

	vGenericRegHealthCheck(r, hHealth)

	routes := r.Routes()
	for _, route := range routes {
		log.Printf("Route registered: %s %s", route.Method, route.Path)
	}

	addr := fmt.Sprintf(":%d", cfg.ListenPort)

	return &httpProc{
		server: http.Server{
			Addr:              addr,
			Handler:           r,
			ReadHeaderTimeout: 2 * time.Second,
		},
		addr: addr,
	}
}

func (p *httpProc) Serve() error {
	log.Printf("Starting HTTP server on %s", p.addr)
	return p.server.ListenAndServe()
}
