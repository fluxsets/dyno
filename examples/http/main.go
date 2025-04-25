package main

import (
	"context"
	"github.com/fluxsets/fleet"
	"github.com/fluxsets/fleet/eventbus"
	"github.com/fluxsets/fleet/option"
	"github.com/fluxsets/fleet/server/http"
	"github.com/fluxsets/fleet/server/subscriber"
	"gocloud.dev/pubsub"
	"gocloud.dev/server/health"
	"log"
	gohttp "net/http"
	"time"
)

type Config struct {
	Addr   string                          `json:"addr"`
	PubSub map[string]eventbus.TopicOption `json:"pubsub"`
}

func main() {
	opt := option.FromFlags()
	opt.Name = "http-example"
	opt.Version = "v0.0.1"
	app := fleet.New(opt, func(ctx context.Context, ft fleet.Fleet) error {
		config := &Config{}
		if err := ft.C().Unmarshal(config); err != nil {
			return err
		}
		//ft.EventBus().Init(fleet.EventBusOption{ExternalTopics: config.PubSub})
		opt := ft.O()
		logger := ft.Logger()
		logger.Info("parsed option", "option", opt)
		logger.Info("parsed config", "config", config)

		var healthChecks []health.Checker
		ft.Hooks().OnStart(func(ctx context.Context) error {
			ft.Logger().Info("on start")
			return nil
		})
		if components, err := ft.ComponentFromProducer(subscriber.NewSubscriberProducer("hello", func(ctx context.Context, msg *pubsub.Message) error {
			logger.Info("recv event", "message", string(msg.Body))
			return nil
		}), fleet.ProduceOption{Instances: 1}); err != nil {
			return err
		} else {
			for _, dep := range components {
				healthChecks = append(healthChecks, dep)
			}
		}
		ft.Hooks().OnStop(func(ctx context.Context) error {
			ft.Logger().Info("on stop")
			return nil
		})
		router := http.NewRouter()
		router.HandleFunc("/", func(rw gohttp.ResponseWriter, r *gohttp.Request) {
			_, _ = rw.Write([]byte("hello"))
		})
		if err := ft.Component(http.NewServer(":9090", router.ServeHTTP, healthChecks, ft.Logger("logger", "http-requestlog"))); err != nil {
			return err
		}

		ft.Hooks().OnStart(func(ctx context.Context) error {
			time.Sleep(1 * time.Second)
			topic, err := ft.EventBus().Topic("hello")
			if err != nil {
				return err
			}
			if err := topic.Send(ctx, &pubsub.Message{
				Body: []byte("hello"),
				Metadata: map[string]string{
					eventbus.KeyName: "hello",
					"from":           ft.O().ID,
				},
			}); err != nil {
				logger.Info("failed to send message", "error", err)
			}
			return nil
		})

		return nil
	})
	err := app.RunE()
	if err != nil {
		log.Fatal(err)
	}
}
