package dyno

import "context"

type Deployment interface {
	ID() string
	Init(do Dyno) error
	Start(ctx context.Context) error
	Stop(ctx context.Context)
}

type DeploymentSet []Deployment

type ServerLike interface {
	Deployment
}
