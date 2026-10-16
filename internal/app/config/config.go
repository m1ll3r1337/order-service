package config

import (
	"io"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/m1ll3r1337/order-service/internal/app/config/section"
	"github.com/m1ll3r1337/order-service/internal/app/constant"
	msentry "github.com/m1ll3r1337/order-service/internal/app/monitor/sentry"
)

type Config struct {
	Repository section.Repository `required:"true"`
	Monitor    section.Monitor    `required:"true"`
	Processor  section.Processor  `required:"true"`
	Client     section.Client     `required:"true"`
}

var Root Config

type LoadArgs struct {
	Output          io.Writer `json:"-"`
	EnableSimpleLog bool
}

func Load(args LoadArgs) {
	zerolog.TimestampFieldName = "timestamp"
	zerolog.MessageFieldName = "message"
	zerolog.TimeFieldFormat = time.RFC3339

	if args.EnableSimpleLog {
		args.Output = zerolog.ConsoleWriter{Out: args.Output}
	}

	log.Logger = createLogger(zerolog.DebugLevel, args.Output)

	log.Debug().Msg("Logger initialized with Debug level")

	if err := godotenv.Load(); err != nil {
		log.Warn().Err(err).Msg("no .env file found")
	}

	if err := envconfig.Process("APP", &Root); err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	lvl, err := zerolog.ParseLevel(Root.Monitor.LogLevel)
	if err != nil {
		log.Warn().Str("log_level", Root.Monitor.LogLevel).Msg("Unknown log level, using debug")
		lvl = zerolog.DebugLevel
	}

	output := args.Output
	w, ok := msentry.Init(Root.Monitor.Sentry, msentry.Options{
		ServiceName: constant.AppName,
		Environment: Root.Monitor.Environment,
		Release:     constant.Version,
	})
	if ok {
		output = zerolog.MultiLevelWriter(args.Output, w)
	}

	log.Logger = createLogger(lvl, output)
	log.Info().Str("log_level", lvl.String()).Msg("Logger re-initialized with config level")
}

func createLogger(level zerolog.Level, output io.Writer) zerolog.Logger {
	return zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Logger()
}
