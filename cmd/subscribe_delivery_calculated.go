package cmd

import (
	"github.com/urfave/cli/v2"

	"github.com/m1ll3r1337/order-service/internal/app/builder"
)

// SubscribeDeliveryCalculated возвращает CLI команду для запуска consumer'а топика order.delivery.calculated.
func SubscribeDeliveryCalculated() *cli.Command {
	return &cli.Command{
		Name:            "subscribe-delivery-calculated",
		Aliases:         []string{"consume-delivery-calculated"},
		Usage:           "Запускает consumer топика order.delivery.calculated",
		Action:          cmdSubscribeDeliveryCalculated,
		HideHelpCommand: true,
	}
}

// cmdSubscribeDeliveryCalculated — handler команды subscribe-delivery-calculated.
// Поднимает consumer Kafka + служебный HTTP (health/metrics/pprof) под общий graceful shutdown.
func cmdSubscribeDeliveryCalculated(cCtx *cli.Context) error {
	app := builder.NewBuilder(cCtx)
	app.BuildConfig()
	app.BuildMonitorOpenTelemetry()
	app.BuildRepoConnPostgres()
	app.BuildRepoOrder()
	app.BuildServiceDelivery()
	app.BuildBrokerKafka()
	app.BuildConsumerDeliveryCalculated()
	app.BuildProcHttpService()

	app.Run()
	return nil
}
