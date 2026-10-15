package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/m1ll3r1337/order-service/cmd"
	"github.com/m1ll3r1337/order-service/internal/app/constant"
	msentry "github.com/m1ll3r1337/order-service/internal/app/monitor/sentry"
)

func main() {
	app := &cli.App{
		Name:    constant.AppName,
		Version: constant.Version,
		Usage:   "Order management service",
		Commands: []*cli.Command{
			cmd.WebServer(),
		},
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "no-json",
				Usage: "Disable JSON logging, use human-readable console output",
			},
		},
	}

	defer msentry.Flush()

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
