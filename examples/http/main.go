package main

import (
	"context"
	"github.com/fluxsets/fleet"
	"github.com/fluxsets/fleet/option"
	"github.com/fluxsets/fleet/server/http"
	"log"
	gohttp "net/http"
	"time"
)

type Config struct {
	Addr string `json:"addr"`
}

func main() {
	opt := option.FromFlags()
	opt.Name = "http-example"
	opt.Version = "v0.0.1"
	app := fleet.New(opt, func(ctx context.Context, ft fleet.Fleet) error {
		config := &Config{}
		if err := ft.Config().Unmarshal(config); err != nil {
			return err
		}
		//ft.EventBus().Init(fleet.EventBusOption{ExternalTopics: config.PubSub})
		opt := ft.Option()
		logger := ft.Logger()
		logger.Info("parsed option", "option", opt)
		logger.Info("parsed config", "config", config)

		ft.Hooks().OnStart(func(ctx context.Context) error {
			ft.Logger().Info("on start")
			return nil
		})

		ft.Hooks().OnStop(func(ctx context.Context) error {
			ft.Logger().Info("on stop")
			return nil
		})
		router := http.NewRouter()
		router.HandleFunc("/", func(rw gohttp.ResponseWriter, r *gohttp.Request) {
			_, _ = rw.Write([]byte("hello"))
		})

		if err := ft.Load(http.NewServer(":9090", router, ft.HealthCheck(), ft.Logger("logger", "http-requestlog"))); err != nil {
			return err
		}

		ft.Hooks().OnStart(func(ctx context.Context) error {
			time.Sleep(1 * time.Second)
			return nil
		})

		return nil
	})
	err := app.RunE()
	if err != nil {
		log.Fatal(err)
	}
}
