package fleet

import (
	"context"
	"gocloud.dev/server/health"
)

type Deployment interface {
	health.Checker
	Name() string
	Init(ft Fleet) error
	Start(ctx context.Context) error
	Stop(ctx context.Context)
}

type DeploymentOptions struct {
	Instances int `json:"instances"` // 实例数
}

func (o *DeploymentOptions) ensureDefaults() {
	if o.Instances == 0 {
		o.Instances = 1
	}
}

type DeploymentProducer func() Deployment

type DeploymentSet []Deployment

type ServerLike interface {
	Deployment
}
