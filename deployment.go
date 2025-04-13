package dyno

import "context"

type Deployment interface {
	ID() string
	Init(do Dyno) error
	Start(ctx context.Context) error
	Stop(ctx context.Context)
}

type DeploymentOptions struct {
	Instances int `json:"instances"` // 实例数
}

type DeploymentFactory func() Deployment

type DeploymentSet []Deployment

type ServerLike interface {
	Deployment
}
