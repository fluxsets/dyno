package dyno

import (
	"context"
	"fmt"
	"go.uber.org/multierr"
	"gocloud.dev/pubsub"
	"gocloud.dev/pubsub/kafkapubsub"
	"gocloud.dev/pubsub/mempubsub"
	"sync"
	"time"
)

type EventBus interface {
	Init(options map[string]TopicOption)
	Subscription(topic string) (*pubsub.Subscription, error)
	Topic(topic string) (*pubsub.Topic, error)
	Close(ctx context.Context) error
}

type TopicOption struct {
	Provider string            `json:"provider"`
	TopicID  string            `json:"topic_id"`
	Kafka    *KafkaTopicOption `json:"kafka,omitempty"`
}

type KafkaTopicOption struct {
	Servers      []string           `json:"servers"`
	Topic        string             `json:"topic"`
	Subscription *KafkaSubscription `json:"subscription"`
}

type KafkaSubscription struct {
	Group string `json:"group"`
}

type pubSub struct {
	mu        sync.RWMutex
	options   map[string]TopicOption
	topics    map[string]*pubsub.Topic
	memTopics map[string]*pubsub.Topic
	memMu     sync.RWMutex
}

func (ps *pubSub) Close(ctx context.Context) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	var multiErr error
	for _, t := range ps.topics {
		if err := t.Shutdown(ctx); err != nil {
			multiErr = multierr.Append(multiErr, err)
		}
	}
	return multiErr
}

func (ps *pubSub) Init(options map[string]TopicOption) {
	for k, o := range options {
		ps.options[k] = o
	}
}

func (ps *pubSub) Subscription(topic string) (*pubsub.Subscription, error) {
	return ps.openSubscription(topic)
}

func (ps *pubSub) addTopic(id string, topic *pubsub.Topic) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.topics[id] = topic
}

func (ps *pubSub) openSubscription(id string) (*pubsub.Subscription, error) {
	o, ok := ps.options[id]
	if ok && o.Provider == "kafka" {
		return ps.openKafkaSubscription(o.Kafka)
	}
	return ps.openMemSubscription(id)
}

func (ps *pubSub) openKafkaSubscription(o *KafkaTopicOption) (*pubsub.Subscription, error) {
	if o.Subscription == nil || o.Subscription.Group == "" {
		return nil, fmt.Errorf("no subscription.group specified")
	}
	config := kafkapubsub.MinimalConfig()
	sub, err := kafkapubsub.OpenSubscription(o.Servers, config, o.Subscription.Group, []string{o.Topic}, nil)
	return sub, err
}

func (ps *pubSub) openMemSubscription(id string) (*pubsub.Subscription, error) {
	topic, _ := ps.openMemTopic(id)
	return mempubsub.NewSubscription(topic, 1*time.Minute), nil
}
func (ps *pubSub) getMemTopic(id string) (*pubsub.Topic, bool) {
	ps.memMu.RLock()
	defer ps.memMu.RUnlock()

	topic, ok := ps.memTopics[id]
	return topic, ok
}

func (ps *pubSub) openMemTopic(id string) (*pubsub.Topic, error) {
	topic, ok := ps.getMemTopic(id)
	if ok {
		return topic, nil
	}
	topic = mempubsub.NewTopic()
	ps.addMemTopic(id, topic)
	return topic, nil
}

func (ps *pubSub) addMemTopic(id string, topic *pubsub.Topic) {
	ps.memMu.Lock()
	defer ps.memMu.Unlock()
	ps.memTopics[id] = topic
}

func (ps *pubSub) getTopic(id string) (*pubsub.Topic, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	topic, ok := ps.topics[id]
	return topic, ok
}

func (ps *pubSub) openKafkaTopic(o *KafkaTopicOption) (*pubsub.Topic, error) {
	config := kafkapubsub.MinimalConfig()
	topic, err := kafkapubsub.OpenTopic(o.Servers, config, o.Topic, nil)
	return topic, err
}

func (ps *pubSub) openTopic(id string) (*pubsub.Topic, error) {
	o, ok := ps.options[id]
	if ok && o.Provider == "kafka" {
		return ps.openKafkaTopic(o.Kafka)
	}

	return ps.openMemTopic(id)
}

func (ps *pubSub) Topic(id string) (*pubsub.Topic, error) {
	topic, ok := ps.getTopic(id)
	if !ok {
		var err error
		topic, err = ps.openTopic(id)
		if err != nil {
			return nil, err
		}
		ps.addTopic(id, topic)
	}

	return topic, nil
}

func newEventBus() EventBus {
	return &pubSub{
		mu:        sync.RWMutex{},
		options:   map[string]TopicOption{},
		topics:    map[string]*pubsub.Topic{},
		memTopics: map[string]*pubsub.Topic{},
		memMu:     sync.RWMutex{},
	}
}
