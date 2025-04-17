package main

import (
	"context"
	"github.com/fluxsets/orbit"
	"github.com/fluxsets/orbit/eventbus"
	"gocloud.dev/pubsub"
	"gocloud.dev/server/health"
	"log"
	"time"
)

type Config struct {
	Addr   string                       `json:"addr"`
	PubSub map[string]orbit.TopicOption `json:"pubsub"`
}

func main() {
	option := orbit.OptionFromFlags()
	option.Name = "cli-example"
	option.Version = "v0.0.1"
	app := orbit.NewApp(option, func(ctx context.Context, ob orbit.Orbit) error {
		config := &Config{}
		if err := ob.Config().Unmarshal(config); err != nil {
			return err
		}
		ob.EventBus().Init(orbit.EventBusOption{ExternalTopics: config.PubSub})

		opt := ob.Option()
		logger := ob.Logger()
		logger.Info("parsed option", "option", opt.String())
		logger.Info("parsed config", "config", config)

		ob.Hooks().OnStart(func(ctx context.Context) error {
			ob.Logger().Info("on start")
			return nil
		})

		healthChecks := []health.Checker{}
		if deployments, err := ob.DeployFromProducer(eventbus.NewSubscriberProducer("hello", func(ctx context.Context, msg *pubsub.Message) error {
			logger.Info("recv event", "message", string(msg.Body))
			return nil
		}), orbit.DeploymentOptions{Instances: 1}); err != nil {
			return err
		} else {
			for _, deployment := range deployments {
				healthChecks = append(healthChecks, deployment)
			}
		}

		ob.Hooks().OnStop(func(ctx context.Context) error {
			ob.Logger().Info("on stop")
			return nil
		})

		if err := ob.Deploy(orbit.NewCommand(func(ctx context.Context) error {
			topic, err := ob.EventBus().Topic("hello")
			if err != nil {
				return err
			}
			if err := topic.Send(ctx, &pubsub.Message{
				Body: []byte("hello"),
				Metadata: map[string]string{
					orbit.KeyName: "hello",
				},
			}); err != nil {
				logger.Info("failed to send message", "error", err)
			}

			logger.Info("command executed successfully")
			time.Sleep(1 * time.Second)

			return nil
		})); err != nil {
			return err
		}

		return nil
	})
	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
