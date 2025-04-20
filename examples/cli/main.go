package main

import (
	"context"
	"github.com/fluxsets/hyper"
	"github.com/fluxsets/hyper/eventbus"
	"github.com/fluxsets/hyper/option"
	"github.com/fluxsets/hyper/server/subscriber"
	"gocloud.dev/pubsub"
	"gocloud.dev/server/health"
	"log"
	"time"
)

type Config struct {
	Addr   string                          `json:"addr"`
	PubSub map[string]eventbus.TopicOption `json:"pubsub"`
}

func main() {
	opt := option.FromFlags()
	opt.Name = "cli-example"
	opt.Version = "v0.0.1"
	app := hyper.New(opt, func(ctx context.Context, hyp hyper.Hyper) error {
		config := &Config{}
		if err := hyp.Config().Unmarshal(config); err != nil {
			return err
		}
		hyp.EventBus().Init(eventbus.Option{ExternalTopics: config.PubSub})

		opt := hyp.Option()
		logger := hyp.Logger()
		logger.Info("parsed option", "option", opt.String())
		logger.Info("parsed config", "config", config)

		hyp.Hooks().OnStart(func(ctx context.Context) error {
			hyp.Logger().Info("on start")
			return nil
		})

		var healthChecks []health.Checker
		if deployments, err := hyp.DeployFromProducer(subscriber.NewSubscriberProducer("hello", func(ctx context.Context, msg *pubsub.Message) error {
			logger.Info("recv event", "message", string(msg.Body))
			return nil
		}), hyper.DeploymentOptions{Instances: 1}); err != nil {
			return err
		} else {
			for _, deployment := range deployments {
				healthChecks = append(healthChecks, deployment)
			}
		}

		hyp.Hooks().OnStop(func(ctx context.Context) error {
			hyp.Logger().Info("on stop")
			return nil
		})

		if err := hyp.DeployCommand(func(ctx context.Context) error {
			topic, err := hyp.EventBus().Topic("hello")
			if err != nil {
				return err
			}
			if err := topic.Send(ctx, &pubsub.Message{
				Body: []byte("hello"),
				Metadata: map[string]string{
					eventbus.KeyName: "hello",
				},
			}); err != nil {
				logger.Info("failed to send message", "error", err)
			}

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
