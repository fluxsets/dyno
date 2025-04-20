package main

import (
	"context"
	"github.com/fluxsets/hyper"
	"github.com/fluxsets/hyper/eventbus"
	"gocloud.dev/pubsub"
	"gocloud.dev/server/health"
	"log"
	"time"
)

type Config struct {
	Addr   string                       `json:"addr"`
	PubSub map[string]hyper.TopicOption `json:"pubsub"`
}

func main() {
	option := hyper.OptionFromFlags()
	option.Name = "cli-example"
	option.Version = "v0.0.1"
	app := hyper.New(option, func(ctx context.Context, hp hyper.Hyper) error {
		config := &Config{}
		if err := hp.Config().Unmarshal(config); err != nil {
			return err
		}
		hp.EventBus().Init(hyper.EventBusOption{ExternalTopics: config.PubSub})

		opt := hp.Option()
		logger := hp.Logger()
		logger.Info("parsed option", "option", opt.String())
		logger.Info("parsed config", "config", config)

		hp.Hooks().OnStart(func(ctx context.Context) error {
			hp.Logger().Info("on start")
			return nil
		})

		healthChecks := []health.Checker{}
		if deployments, err := hp.DeployFromProducer(eventbus.NewSubscriberProducer("hello", func(ctx context.Context, msg *pubsub.Message) error {
			logger.Info("recv event", "message", string(msg.Body))
			return nil
		}), hyper.DeploymentOptions{Instances: 1}); err != nil {
			return err
		} else {
			for _, deployment := range deployments {
				healthChecks = append(healthChecks, deployment)
			}
		}

		hp.Hooks().OnStop(func(ctx context.Context) error {
			hp.Logger().Info("on stop")
			return nil
		})

		if err := hp.Deploy(hyper.NewCommand(func(ctx context.Context) error {
			topic, err := hp.EventBus().Topic("hello")
			if err != nil {
				return err
			}
			if err := topic.Send(ctx, &pubsub.Message{
				Body: []byte("hello"),
				Metadata: map[string]string{
					hyper.KeyName: "hello",
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
	err := app.RunE()
	if err != nil {
		log.Fatal(err)
	}
}
