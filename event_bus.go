package dyno

import "gocloud.dev/pubsub"

type EventBus interface {
	Subscription(topic string) *pubsub.Subscription
	Topic(topic string) *pubsub.Topic
}

type pubSubEventBus struct {
}

func (pb *pubSubEventBus) Subscription(topic string) *pubsub.Subscription {
}

func (pb *pubSubEventBus) Topic(topic string) *pubsub.Topic {
}

var _ EventBus = (*pubSubEventBus)(nil)

func NewEventBus() EventBus {
	return &pubSubEventBus{}
}
