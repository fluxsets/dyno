package main

import (
	"context"
	"github.com/fluxsets/fleet"
	"github.com/fluxsets/fleet/eventbus"
	"github.com/fluxsets/fleet/option"
	"github.com/fluxsets/fleet/server/subscriber"
	"gocloud.dev/pubsub"
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
	app := fleet.New(opt, func(ctx context.Context, ft fleet.Fleet) error {
		config := &Config{}
		if err := ft.C().Unmarshal(config); err != nil {
			return err
		}
		ft.EventBus().Init(eventbus.Option{ExternalTopics: config.PubSub})

		opt := ft.O()
		logger := ft.Logger()
		logger.Info("parsed option", "option", opt.String())
		logger.Info("parsed config", "config", config)

		ft.Hooks().OnStart(func(ctx context.Context) error {
			ft.Logger().Info("on start")
			return nil
		})

		if _, err := ft.ComponentFromProducer(subscriber.NewSubscriberProducer("hello", func(ctx context.Context, msg *pubsub.Message) error {
			logger.Info("recv event", "message", string(msg.Body))
			return nil
		}, 1)); err != nil {
			return err
		}

		ft.Hooks().OnStop(func(ctx context.Context) error {
			ft.Logger().Info("on stop")
			return nil
		})

		if err := ft.Command(func(ctx context.Context) error {
			topic, err := ft.EventBus().Topic("hello")
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
