package builder

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"

	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	"github.com/m1ll3r1337/order-service/internal/app/config"
	rhandler "github.com/m1ll3r1337/order-service/internal/app/handler/http"
	rhealth "github.com/m1ll3r1337/order-service/internal/app/handler/http/health"
	"github.com/m1ll3r1337/order-service/internal/app/processor"
	rprocessor "github.com/m1ll3r1337/order-service/internal/app/processor/http"
	rcpostgres "github.com/m1ll3r1337/order-service/internal/app/repository/conn/postgres"
)

type Builder struct {
	cCtx *cli.Context
	ctx  context.Context
	wg   sync.WaitGroup
	err  error
	cfg  config.Config

	chErrors chan error

	connPostgres *rcpostgres.Client

	healthHandler rhandler.Health

	processors []processor.Processor
}

func NewBuilder(cCtx *cli.Context) *Builder {
	b := Builder{
		cCtx:     cCtx,
		chErrors: make(chan error, 4096),
	}

	ctx, cancelFunc := context.WithCancel(context.Background())
	b.ctx = ctx

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP)

	go b.waitForSignal(sig, cancelFunc)
	go b.printErrors()

	b.healthHandler = rhealth.NewHandler()

	return &b
}

func (b *Builder) BuildConfig() {
	b.exec(func(b *Builder) {
		b.buildConfig()
	})
}

func (b *Builder) Run() {
	if b.ctx.Err() != nil {
		log.Info().Msg("Shutdown during initialization")
		return
	}

	if b.err != nil {
		log.Fatal().Err(b.err).Msg("Failed to initialize application")
	}

	log.Info().Msg("Application initialized")
	defer log.Info().Msg("Application completed")

	for _, p := range b.processors {
		p.StartAsync(b.ctx, &b.wg)
	}

	b.wg.Wait()
}

////////////////////////////////////////////////////////////////////////////////
///// REPOSITORY CONNECTIONS ///////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (b *Builder) BuildRepoConnPostgres() {
	b.exec(func(b *Builder) {
		conn, err := rcpostgres.NewClient(b.ctx, b.cfg.Repository.Postgres)
		if err != nil {
			b.err = fmt.Errorf("failed to create postgres conn: %w", err)
			return
		}

		b.connPostgres = conn
	})
}

////////////////////////////////////////////////////////////////////////////////
///// PROCESSORS ///////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (b *Builder) BuildProcHttp() {
	b.exec(func(b *Builder) {
		pr := rprocessor.NewHTTP(
			b.healthHandler,
			b.cfg.Processor.WebServer,
		)
		b.processors = append(b.processors, pr)
	}, b.healthHandler)
}

////////////////////////////////////////////////////////////////////////////////
///// PRIVATE //////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (b *Builder) buildConfig() {
	args := config.LoadArgs{
		Output:          b.cCtx.App.Writer,
		EnableSimpleLog: b.cCtx.Bool("no-json"),
	}

	config.Load(args)
	b.cfg = config.Root
}

func (b *Builder) waitForSignal(sig chan os.Signal, cancelFunc func()) {
	s := <-sig

	log.Info().Str("signal", s.String()).Msg("Shutdown is requested")
	cancelFunc()
}

func (b *Builder) printErrors() {
	for err := range b.chErrors {
		log.Error().Err(err).Msg("Got new error")
	}
}

func (b *Builder) exec(cb func(b *Builder), requiredArgs ...any) {
	if b.err != nil || b.ctx.Err() != nil {
		return
	}

	for i, requiredArg := range requiredArgs {
		rv := reflect.ValueOf(requiredArg)
		if !rv.IsValid() {
			b.err = fmt.Errorf("BUG: required argument #%d is nil (check dependencies)", i)
			return
		}
		if rv.Type().Kind() == reflect.Struct || !rv.IsZero() {
			continue
		}
		b.err = fmt.Errorf("BUG: required %s, but empty", rv.Type().String())
		return
	}

	cb(b)
}
