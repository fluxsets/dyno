package main

import (
	"context"
	"github.com/fluxsets/fleet"
	"github.com/fluxsets/fleet/option"
	"log"
	"time"
)

type Config struct {
	Addr string `json:"addr"`
}

func main() {
	op := option.FromFlags()
	op.Name = "cli-example"
	op.Version = "v0.0.1"
	app := fleet.New(op, func(ctx context.Context, ft fleet.Fleet) error {
		config := &Config{}
		if err := ft.Configurer().Unmarshal(config); err != nil {
			return err
		}

		opt := ft.Option()
		logger := ft.Logger()
		logger.Info("parsed option", "option", opt.String())
		logger.Info("parsed config", "config", config)

		ft.Hooks().OnStart(func(ctx context.Context) error {
			ft.Logger().Info("on start")
			return nil
		})

		ft.Hooks().OnStop(func(ctx context.Context) error {
			ft.Logger().Info("on stop")
			return nil
		})

		if err := ft.Command(func(ctx context.Context) error {
			logger.Info("command executed successfully")
			time.Sleep(1 * time.Second)
			return nil
		}); err != nil {
			return err
		}

		return nil
	})
	err := app.RunE()
	if err != nil {
		log.Fatal(err)
	}
}
