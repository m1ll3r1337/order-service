package rprocessor

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"

	"github.com/m1ll3r1337/order-service/internal/app/config/section"
	rhandler "github.com/m1ll3r1337/order-service/internal/app/handler/http"
	"github.com/m1ll3r1337/order-service/internal/app/processor"
	"github.com/m1ll3r1337/order-service/internal/app/util"
	"github.com/m1ll3r1337/order-service/internal/pkg/http/httph"
	"github.com/m1ll3r1337/order-service/internal/pkg/http/mzerolog"
)

type httpProc struct {
	server http.Server
	addr   string
}

func NewHTTP(hHealth rhandler.Health, cfg section.ProcessorWebServer) processor.Processor {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(
		adaptRequestMiddleware(httph.NewErrorMiddleware()),
		mzerolog.NewMiddleware(mzerolog.WithSkipper(util.IsFilteredHttpRoute)),
		gin.Recovery(),
	)

	router.NoRoute(handleNotFound)
	vGenericRegHealthCheck(router, hHealth)

	routes := router.Routes()
	for _, route := range routes {
		log.Info().Str("method", route.Method).Str("path", route.Path).Msg("Route registered")
	}

	p := httpProc{addr: fmt.Sprintf(":%d", cfg.ListenPort)}
	p.server.Handler = router

	return &p
}

func (p *httpProc) StartAsync(ctx context.Context, wg *sync.WaitGroup) {
	lc := net.ListenConfig{}

	l, err := lc.Listen(ctx, "tcp", p.addr)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to start listening TCP addr")
		return
	}

	log.Info().Str("listen_addr", p.addr).Msg("Listening of TCP addr for HTTP server has been started")

	go p.serve(l)

	processor.WatchForShutdown(ctx, wg, processor.CloserFunc(l.Close))
	processor.WatchForShutdown(ctx, wg, processor.NewCloserContextFunc(p.server.Shutdown, context.Background(), 5*time.Second))
}

func (p *httpProc) serve(l net.Listener) {
	_ = p.server.Serve(l)
}

func adaptRequestMiddleware(m httph.Middleware) gin.HandlerFunc {
	return func(c *gin.Context) {
		m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c.Request = r
			c.Next()
		})).ServeHTTP(c.Writer, c.Request)
	}
}
