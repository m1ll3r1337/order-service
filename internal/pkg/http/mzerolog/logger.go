package mzerolog

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/m1ll3r1337/order-service/internal/pkg/http/httph"
)

type middleware struct {
	log zerolog.Logger

	fromOptions struct {
		skipper func(r *http.Request) bool
	}
}

func (m *middleware) Callback(c *gin.Context) {
	const (
		tailSuccess = " finished with no error"
		tailFail    = " finished (or aborted) with error"
	)

	timeStart := time.Now()

	c.Next()

	err := httph.ErrorGet(c.Request)

	execTime := time.Since(timeStart)

	if m.fromOptions.skipper(c.Request) {
		return
	}

	var mb strings.Builder
	mb.WriteString(c.Request.Method)
	mb.WriteString(" ")
	mb.WriteString(c.Request.RequestURI)

	var ev *zerolog.Event
	if err != nil {
		mb.WriteString(tailFail)
		ev = m.log.Error()
	} else {
		mb.WriteString(tailSuccess)
		ev = m.log.Debug()
	}

	ev.Err(err).
		Ctx(c.Request.Context()).
		Dur("exec_time", execTime).
		Str("client_ip", c.ClientIP()).
		Msg(mb.String())
}

func NewMiddleware(opts ...Option) gin.HandlerFunc {
	m := &middleware{
		log: log.Logger,
	}

	m.fromOptions.skipper = defaultSkipper

	for _, opt := range opts {
		opt(m)
	}

	return m.Callback
}

func defaultSkipper(r *http.Request) bool {
	return false
}
