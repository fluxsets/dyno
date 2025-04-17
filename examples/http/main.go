package main

import (
	"context"
	"github.com/fluxsets/orbit"
	"github.com/fluxsets/orbit/eventbus"
	"github.com/fluxsets/orbit/server/http"
	"gocloud.dev/pubsub"
	"gocloud.dev/server/health"
	"log"
	gohttp "net/http"
)

type Config struct {
	Addr   string                       `json:"addr"`
	PubSub map[string]orbit.TopicOption `json:"pubsub"`
}

func main() {
	option := orbit.OptionFromFlags()
	option.Name = "http-example"
	option.Version = "v0.0.1"
	app := orbit.New(option, func(ctx context.Context, ob orbit.Orbit) error {
		config := &Config{}
		if err := ob.Config().Unmarshal(config); err != nil {
			return err
		}
		ob.EventBus().Init(orbit.EventBusOption{ExternalTopics: config.PubSub})
		opt := ob.Option()
		logger := ob.Logger()
		logger.Info("parsed option", "option", opt)
		logger.Info("parsed config", "config", config)

		var healthChecks []health.Checker
		ob.Hooks().OnStart(func(ctx context.Context) error {
			ob.Logger().Info("on start")
			return nil
		})
		if deployments, err := ob.DeployFromProducer(eventbus.NewSubscriberProducer("hello", func(ctx context.Context, msg *pubsub.Message) error {
			logger.Info("recv event", "message", string(msg.Body))
			return nil
		}), orbit.DeploymentOptions{Instances: 1}); err != nil {
			return err
		} else {
			for _, dep := range deployments {
				healthChecks = append(healthChecks, dep)
			}
		}
		ob.Hooks().OnStop(func(ctx context.Context) error {
			ob.Logger().Info("on stop")
			return nil
		})
		router := http.NewRouter()
		router.HandleFunc("/", func(rw gohttp.ResponseWriter, r *gohttp.Request) {
			_, _ = rw.Write([]byte("hello"))
		})
		if err := ob.Deploy(http.NewServer(":9090", router.ServeHTTP, healthChecks, ob.Logger("logger", "http-requestlog"))); err != nil {
			return err
		}
		topic, err := ob.EventBus().Topic("hello")
		if err != nil {
			return err
		}
		if err := topic.Send(ctx, &pubsub.Message{
			Body: []byte("hello"),
			Metadata: map[string]string{
				orbit.KeyName: "hello",
				"from":        ob.Option().ID,
			},
		}); err != nil {
			logger.Info("failed to send message", "error", err)
		}

		return nil
	})
	err := app.RunE()
	if err != nil {
		log.Fatal(err)
	}
}
