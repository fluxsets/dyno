package main

import (
	"context"
	"github.com/fluxsets/dyno"
	"github.com/fluxsets/dyno/eventbus"
	"github.com/fluxsets/dyno/server/http"
	"gocloud.dev/pubsub"
	"gocloud.dev/server/health"
	"log"
	gohttp "net/http"
)

type Config struct {
	Addr   string                      `json:"addr"`
	PubSub map[string]dyno.TopicOption `json:"pubsub"`
}

func main() {
	option := dyno.OptionFromFlags()
	option.Name = "http-example"
	option.Version = "v0.0.1"
	app := dyno.NewApp(option, func(ctx context.Context, do dyno.Dyno) error {
		config := &Config{}
		if err := do.Config().Unmarshal(config); err != nil {
			return err
		}
		do.EventBus().Init(dyno.EventBusOption{BridgeTopics: config.PubSub})
		opt := do.Option()
		logger := do.Logger()
		logger.Info("parsed option", "option", opt)
		logger.Info("parsed config", "config", config)

		healthChecks := []health.Checker{}

		do.Hooks().OnStart(func(ctx context.Context) error {
			do.Logger().Info("on start")
			return nil
		})
		if deps, err := do.DeployFromProducer(eventbus.NewSubscriberProducer("hello", func(ctx context.Context, msg *pubsub.Message) error {
			logger.Info("recv event", "message", string(msg.Body))
			return nil
		}), dyno.DeploymentOptions{Instances: 1}); err != nil {
			return err
		} else {
			for _, dep := range deps {
				healthChecks = append(healthChecks, dep)
			}
		}
		do.Hooks().OnStop(func(ctx context.Context) error {
			do.Logger().Info("on stop")
			return nil
		})
		router := http.NewRouter()
		router.HandleFunc("/hello", func(rw gohttp.ResponseWriter, r *gohttp.Request) {
			_, _ = rw.Write([]byte("hello"))
		})
		if err := do.Deploy(http.NewServer(":9090", router.ServeHTTP, healthChecks)); err != nil {
			return err
		}
		topic, err := do.EventBus().Topic("hello")
		if err != nil {
			return err
		}
		if err := topic.Send(ctx, &pubsub.Message{
			Body: []byte("hello"),
		}); err != nil {
			logger.Info("failed to send message", "error", err)
		}

		return nil
	})
	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
