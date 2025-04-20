package main

import (
	"context"
	"github.com/fluxsets/hyper"
	"github.com/fluxsets/hyper/eventbus"
	"github.com/fluxsets/hyper/option"
	"github.com/fluxsets/hyper/server/http"
	"github.com/fluxsets/hyper/server/subscriber"
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
	app := hyper.New(opt, func(ctx context.Context, hyp hyper.Hyper) error {
		config := &Config{}
		if err := hyp.Config().Unmarshal(config); err != nil {
			return err
		}
		//hyp.EventBus().Init(hyper.EventBusOption{ExternalTopics: config.PubSub})
		opt := hyp.Option()
		logger := hyp.Logger()
		logger.Info("parsed option", "option", opt)
		logger.Info("parsed config", "config", config)

		var healthChecks []health.Checker
		hyp.Hooks().OnStart(func(ctx context.Context) error {
			hyp.Logger().Info("on start")
			return nil
		})
		if deployments, err := hyp.DeployFromProducer(subscriber.NewSubscriberProducer("hello", func(ctx context.Context, msg *pubsub.Message) error {
			logger.Info("recv event", "message", string(msg.Body))
			return nil
		}), hyper.DeploymentOptions{Instances: 1}); err != nil {
			return err
		} else {
			for _, dep := range deployments {
				healthChecks = append(healthChecks, dep)
			}
		}
		hyp.Hooks().OnStop(func(ctx context.Context) error {
			hyp.Logger().Info("on stop")
			return nil
		})
		router := http.NewRouter()
		router.HandleFunc("/", func(rw gohttp.ResponseWriter, r *gohttp.Request) {
			_, _ = rw.Write([]byte("hello"))
		})
		if err := hyp.Deploy(http.NewServer(":9090", router.ServeHTTP, healthChecks, hyp.Logger("logger", "http-requestlog"))); err != nil {
			return err
		}

		hyp.Hooks().OnStart(func(ctx context.Context) error {
			time.Sleep(1 * time.Second)
			topic, err := hyp.EventBus().Topic("hello")
			if err != nil {
				return err
			}
			if err := topic.Send(ctx, &pubsub.Message{
				Body: []byte("hello"),
				Metadata: map[string]string{
					eventbus.KeyName: "hello",
					"from":           hyp.Option().ID,
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
