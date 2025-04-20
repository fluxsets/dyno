package main

import (
	"context"
	"github.com/fluxsets/hyper"
	"github.com/fluxsets/hyper/eventbus"
	"github.com/fluxsets/hyper/server/http"
	"gocloud.dev/pubsub"
	"gocloud.dev/server/health"
	"log"
	gohttp "net/http"
)

type Config struct {
	Addr   string                       `json:"addr"`
	PubSub map[string]hyper.TopicOption `json:"pubsub"`
}

func main() {
	option := hyper.OptionFromFlags()
	option.Name = "http-example"
	option.Version = "v0.0.1"
	app := hyper.New(option, func(ctx context.Context, hp hyper.Hyper) error {
		config := &Config{}
		if err := hp.Config().Unmarshal(config); err != nil {
			return err
		}
		//hp.EventBus().Init(hyper.EventBusOption{ExternalTopics: config.PubSub})
		opt := hp.Option()
		logger := hp.Logger()
		logger.Info("parsed option", "option", opt)
		logger.Info("parsed config", "config", config)

		var healthChecks []health.Checker
		hp.Hooks().OnStart(func(ctx context.Context) error {
			hp.Logger().Info("on start")
			return nil
		})
		if deployments, err := hp.DeployFromProducer(eventbus.NewSubscriberProducer("hello", func(ctx context.Context, msg *pubsub.Message) error {
			logger.Info("recv event", "message", string(msg.Body))
			return nil
		}), hyper.DeploymentOptions{Instances: 1}); err != nil {
			return err
		} else {
			for _, dep := range deployments {
				healthChecks = append(healthChecks, dep)
			}
		}
		hp.Hooks().OnStop(func(ctx context.Context) error {
			hp.Logger().Info("on stop")
			return nil
		})
		router := http.NewRouter()
		router.HandleFunc("/", func(rw gohttp.ResponseWriter, r *gohttp.Request) {
			_, _ = rw.Write([]byte("hello"))
		})
		if err := hp.Deploy(http.NewServer(":9090", router.ServeHTTP, healthChecks, hp.Logger("logger", "http-requestlog"))); err != nil {
			return err
		}
		topic, err := hp.EventBus().Topic("hello")
		if err != nil {
			return err
		}
		if err := topic.Send(ctx, &pubsub.Message{
			Body: []byte("hello"),
			Metadata: map[string]string{
				hyper.KeyName: "hello",
				"from":        hp.Option().ID,
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
