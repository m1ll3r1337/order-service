package builder

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"sync"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"

	ccatalog "github.com/m1ll3r1337/order-service/internal/app/client/catalog"
	cgrpc "github.com/m1ll3r1337/order-service/internal/app/client/catalog/grpc"
	"github.com/m1ll3r1337/order-service/internal/app/config"
	"github.com/m1ll3r1337/order-service/internal/app/constant"
	"github.com/m1ll3r1337/order-service/internal/app/entity"
	rhandler "github.com/m1ll3r1337/order-service/internal/app/handler/http"
	rhealth "github.com/m1ll3r1337/order-service/internal/app/handler/http/health"
	horder "github.com/m1ll3r1337/order-service/internal/app/handler/http/order"
	mmetric "github.com/m1ll3r1337/order-service/internal/app/monitor/metric"
	"github.com/m1ll3r1337/order-service/internal/app/processor"
	rprocessor "github.com/m1ll3r1337/order-service/internal/app/processor/http"
	"github.com/m1ll3r1337/order-service/internal/app/processor/monitor"
	"github.com/m1ll3r1337/order-service/internal/app/repository"
	rcpostgres "github.com/m1ll3r1337/order-service/internal/app/repository/conn/postgres"
	porder "github.com/m1ll3r1337/order-service/internal/app/repository/order"
	"github.com/m1ll3r1337/order-service/internal/app/service"
	sorder "github.com/m1ll3r1337/order-service/internal/app/service/order"
	"github.com/m1ll3r1337/order-service/internal/pkg/broker"
	"github.com/m1ll3r1337/order-service/internal/pkg/broker/codec"
)

type Builder struct {
	cCtx *cli.Context
	ctx  context.Context
	wg   sync.WaitGroup
	err  error
	cfg  config.Config

	chErrors chan error

	connPostgres *rcpostgres.Client

	orderRepo repository.Order

	orderService service.Order

	healthHandler rhandler.Health
	orderHandler  rhandler.Order

	catalogV1Client ccatalog.Client
	catalogGrpcConn *grpc.ClientConn

	kafkaClient     *broker.KafkaClient
	busOrderCreated broker.Bus[entity.EventOrderCreated]

	otelServiceName string

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
///// REPOSITORIES /////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (b *Builder) BuildRepoOrder() {
	b.exec(func(b *Builder) {
		b.orderRepo = porder.NewRepo(b.connPostgres)
	}, b.connPostgres)
}

func (b *Builder) BuildClientGrpcCatalogV1() {
	b.exec(func(b *Builder) {
		client, conn, err := cgrpc.NewClient(b.cfg.Client.Catalog.GrpcAddress)
		if err != nil {
			b.err = fmt.Errorf("failed to create grpc catalog client: %w", err)
			return
		}
		if err := client.Ping(b.ctx); err != nil {
			_ = conn.Close()
			b.err = fmt.Errorf("failed to ping grpc catalog client: %w", err)
			return
		}

		b.catalogV1Client = client
		b.catalogGrpcConn = conn

		processor.WatchForShutdown(b.ctx, &b.wg, b.catalogGrpcConn)
	})
}

////////////////////////////////////////////////////////////////////////////////
///// BROKER ///////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (b *Builder) BuildBrokerKafka() {
	b.exec(func(b *Builder) {
		cfg := b.cfg.Broker.Kafka

		kafka, err := broker.NewKafkaClient(broker.KafkaConfig{
			Addresses:     b.cfg.Broker.Kafka.Addresses,
			ConsumerGroup: b.cfg.Broker.Kafka.ConsumerGroup,
			ClientID:      b.cfg.Broker.Kafka.ClientID,
		})
		if err != nil {
			b.err = fmt.Errorf("failed to create kafka client: %w", err)
			return
		}

		b.kafkaClient = kafka

		topic := cfg.ModelOrder.Created.Topic
		group := broker.Coalesce(cfg.ModelOrder.Created.ConsumerGroup, cfg.ConsumerGroup)

		bus, err := broker.NewBus[entity.EventOrderCreated](kafka, codec.NewCodecJson[entity.EventOrderCreated](), topic, group)
		if err != nil {
			b.err = fmt.Errorf("failed to create event.order.created bus: %w", err)
			return
		}

		b.busOrderCreated = bus

		processor.WatchForShutdown(b.ctx, &b.wg, processor.CloserFunc(kafka.Close))
	})
}

////////////////////////////////////////////////////////////////////////////////
///// SERVICES /////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (b *Builder) BuildServiceOrder() {
	b.exec(func(b *Builder) {
		orderMetrics, err := mmetric.NewPrometheusOrder(prometheus.DefaultRegisterer)
		if err != nil {
			b.err = fmt.Errorf("init order metrics: %w", err)
			return
		}
		counts, err := b.orderRepo.CountByStatus(b.ctx)
		if err != nil {
			b.err = fmt.Errorf("init order status gauge: %w", err)
			return
		}
		for status, count := range counts {
			orderMetrics.Status().Set(status, float64(count))
		}
		b.orderService = sorder.NewService(b.orderRepo, b.catalogV1Client, orderMetrics, b.busOrderCreated)
	}, b.orderRepo, b.catalogV1Client, b.busOrderCreated)
}

////////////////////////////////////////////////////////////////////////////////
///// HANDLERS /////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (b *Builder) BuildHandlerOrder() {
	b.exec(func(b *Builder) {
		b.orderHandler = horder.NewHandler(b.orderService)
	}, b.orderService)
}

////////////////////////////////////////////////////////////////////////////////
///// PROCESSORS ///////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func (b *Builder) BuildProcHttp() {
	b.exec(func(b *Builder) {
		pr := rprocessor.NewHTTP(
			b.otelServiceName,
			b.healthHandler,
			b.orderHandler,
			b.cfg.Processor.WebServer,
		)
		b.processors = append(b.processors, pr)
	}, b.healthHandler, b.orderHandler)
}

func (b *Builder) BuildMonitorOpenTelemetry() {
	cfg := b.cfg.Monitor.OpenTelemetry
	if !cfg.Enabled {
		log.Warn().Msg("OpenTelemetry is disabled by config")
		return
	}

	b.exec(func(b *Builder) {
		proc, err := monitor.NewOpenTelemetryController(b.ctx, b.cfg.Monitor.Environment, cfg)
		if err != nil {
			b.err = fmt.Errorf("init OpenTelemetry: %w", err)
			return
		}
		b.processors = append(b.processors, proc)
		b.otelServiceName = constant.AppName
	})
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
