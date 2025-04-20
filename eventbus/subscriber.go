package eventbus

import (
	"context"
	"github.com/fluxsets/hyper"
	"gocloud.dev/pubsub"
	"log/slog"
)

type HandlerFunc func(ctx context.Context, msg *pubsub.Message) error

const TopicSep = "::"

type TopicURI string

//func NewTopic(namespace string, topic string) TopicURI {
//	return TopicURI(namespace + TopicSep + topic)
//}

func (t TopicURI) String() string {
	return string(t)
}

//func (t TopicURI) Namespace() string {
//	return strings.Split(string(t), TopicSep)[0]
//}
//
//func (t TopicURI) EventName() string {
//	return strings.Split(string(t), TopicSep)[1]
//}

type Subscriber struct {
	topic   TopicURI
	hyper   hyper.Hyper
	handler HandlerFunc
	subs    *pubsub.Subscription
	logger  *slog.Logger
	hyper.HealthCheck
}

func NewSubscriber(topic TopicURI, h HandlerFunc) *Subscriber {
	return &Subscriber{
		topic:   topic,
		handler: h,
	}
}

func NewSubscriberProducer(topic TopicURI, h HandlerFunc) hyper.DeploymentProducer {
	return func() hyper.Deployment {
		return NewSubscriber(topic, h)
	}
}

func (s *Subscriber) Name() string {
	return "subscriber@" + s.topic.String()
}

func (s *Subscriber) Init(hp hyper.Hyper) error {
	s.hyper = hp
	s.logger = hp.Logger("logger", s.Name())
	return nil
}

func (s *Subscriber) Start(ctx context.Context) error {
	s.logger.Info("starting subscriber")
	s.SetHealthy(true)

	var err error
	s.subs, err = s.hyper.EventBus().Subscription(s.topic.String())
	for {
		var msg *pubsub.Message
		msg, err = s.subs.Receive(ctx)
		if err != nil {
			break
		}
		if err := s.handler(ctx, msg); err != nil {
			s.hyper.Logger().Error("message handle error", "topic", s.topic, "error", err)
		}
		msg.Ack()
	}
	s.SetHealthy(false)
	return err
}

func (s *Subscriber) Stop(ctx context.Context) {
	if err := s.subs.Shutdown(ctx); err != nil {
		s.hyper.Logger().Error("subscription shutdown error", "topic", s.topic, "error", err)
	}
	s.logger.Info("subscriber shut down")
}

var _ hyper.ServerLike = new(Subscriber)
