package main

import (
	"context"
	"github.com/fluxsets/dyno"
	"github.com/fluxsets/dyno/eventbus"
	"gocloud.dev/pubsub"
	"log"
	"time"
)

type Config struct {
	Addr   string                      `json:"addr"`
	PubSub map[string]dyno.TopicOption `json:"pubsub"`
}

func main() {
	option := dyno.OptionFromFlags()
	option.Name = "cli-example"
	option.Version = "v0.0.1"
	app := dyno.NewApp(option, func(ctx context.Context, do dyno.Dyno) error {
		config := &Config{}
		if err := do.Config().Unmarshal(config); err != nil {
			return err
		}
		do.EventBus().Init(config.PubSub)

		opt := do.Option()
		logger := do.Logger()
		logger.Info("parsed option", "option", opt.String())
		logger.Info("parsed config", "config", config)

		do.Hooks().OnStart(func(ctx context.Context) error {
			do.Logger().Info("on start")
			return nil
		})

		if err := do.DeployFromProducer(eventbus.NewSubscriberProducer("hello", func(ctx context.Context, msg *pubsub.Message) error {
			logger.Info("recv event", "message", string(msg.Body))
			return nil
		}), dyno.DeploymentOptions{Instances: 1}); err != nil {
			return err
		}

		do.Hooks().OnStop(func(ctx context.Context) error {
			do.Logger().Info("on stop")
			return nil
		})

		if err := do.Deploy(dyno.Command(func(ctx context.Context) error {
			topic, err := do.EventBus().Topic("hello")
			if err != nil {
				return err
			}
			if err := topic.Send(ctx, &pubsub.Message{
				Body: []byte("hello"),
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
