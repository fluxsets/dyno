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
	app := fleet.New(opt, func(ctx context.Context, flt fleet.Fleet) error {
		config := &Config{}
		if err := flt.Config().Unmarshal(config); err != nil {
			return err
		}
		//flt.EventBus().Init(fleet.EventBusOption{ExternalTopics: config.PubSub})
		opt := flt.Option()
		logger := flt.Logger()
		logger.Info("parsed option", "option", opt)
		logger.Info("parsed config", "config", config)

		var healthChecks []health.Checker
		flt.Hooks().OnStart(func(ctx context.Context) error {
			flt.Logger().Info("on start")
			return nil
		})
		if deployments, err := flt.DeployFromProducer(subscriber.NewSubscriberProducer("hello", func(ctx context.Context, msg *pubsub.Message) error {
			logger.Info("recv event", "message", string(msg.Body))
			return nil
		}), fleet.DeploymentOptions{Instances: 1}); err != nil {
			return err
		} else {
			for _, dep := range deployments {
				healthChecks = append(healthChecks, dep)
			}
		}
		flt.Hooks().OnStop(func(ctx context.Context) error {
			flt.Logger().Info("on stop")
			return nil
		})
		router := http.NewRouter()
		router.HandleFunc("/", func(rw gohttp.ResponseWriter, r *gohttp.Request) {
			_, _ = rw.Write([]byte("hello"))
		})
		if err := flt.Deploy(http.NewServer(":9090", router.ServeHTTP, healthChecks, flt.Logger("logger", "http-requestlog"))); err != nil {
			return err
		}

		flt.Hooks().OnStart(func(ctx context.Context) error {
			time.Sleep(1 * time.Second)
			topic, err := flt.EventBus().Topic("hello")
			if err != nil {
				return err
			}
			if err := topic.Send(ctx, &pubsub.Message{
				Body: []byte("hello"),
				Metadata: map[string]string{
					eventbus.KeyName: "hello",
					"from":           flt.Option().ID,
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
