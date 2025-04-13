package eventbus

import (
	"context"
	"github.com/fluxsets/dyno"
	"strings"
)

const TopicSep = "::"

type TopicURI string

func NewTopic(namespace string, topic string) TopicURI {
	return TopicURI(namespace + TopicSep + topic)
}

func (t TopicURI) String() string {
	return string(t)
}
func (t TopicURI) Namespace() string {
	return strings.Split(string(t), TopicSep)[0]
}

func (t TopicURI) EventName() string {
	return strings.Split(string(t), TopicSep)[1]
}

type Subscriber struct {
	topic TopicURI
}

func (s Subscriber) ID() string {
	return "subscriber:" + s.topic.String()
}

func (s Subscriber) Init(do dyno.Dyno) error {
	//TODO implement me
	panic("implement me")
}

func (s Subscriber) Start(ctx context.Context) error {
	//TODO implement me
	panic("implement me")
}

func (s Subscriber) Stop(ctx context.Context) {
	//TODO implement me
	panic("implement me")
}

var _ dyno.ServerLike = new(Subscriber)
