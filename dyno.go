package dyno

import (
	"context"
	"github.com/oklog/run"
	"log/slog"
)

type Dyno interface {
	Option() Option
	Context() context.Context
	DeployWithOptions(factory DeploymentFactory, options DeploymentOptions) error
	Deploy(deps ...Deployment) error
	Run() error
	EventBus() EventBus
	Hooks() Hooks
	Logger(args ...any) *slog.Logger
}

type dyno struct {
	ctx      context.Context
	o        Option
	runG     *run.Group
	eventBus EventBus
	hooks    *hooks
	logger   *slog.Logger
}

func (do *dyno) DeployWithOptions(factory DeploymentFactory, options DeploymentOptions) error {
	var deps []Deployment
	for i := 0; i < options.Instances; i++ {
		dep := factory()
		deps = append(deps, dep)
	}
	return do.Deploy(deps...)
}

func (do *dyno) Logger(args ...any) *slog.Logger {
	return do.logger.With(args...)
}

func (do *dyno) Context() context.Context {
	return do.ctx
}

func (do *dyno) Option() Option {
	return do.o
}

func (do *dyno) Deploy(deps ...Deployment) error {
	for _, dep := range deps {
		ctx, cancel := context.WithCancel(context.Background())
		if err := dep.Init(do); err != nil {
			cancel()
			return err
		}
		do.runG.Add(func() error {
			return dep.Start(ctx)
		}, func(err error) {
			dep.Stop(ctx)
			cancel()
		})
	}
	return nil
}

func (do *dyno) Run() error {
	for _, fn := range do.hooks.onStarts {
		if err := fn(do.ctx); err != nil {
			return err
		}
	}
	err := do.runG.Run()
	for _, fn := range do.hooks.onStops {
		if err := fn(do.ctx); err != nil {
			do.logger.ErrorContext(do.ctx, "post stop func called error", "error", err)
		}
	}
	return err
}

func (do *dyno) EventBus() EventBus {
	return do.eventBus
}

func (do *dyno) Hooks() Hooks {
	return do.hooks
}

func newDyno(o Option) Dyno {
	return &dyno{
		o:      o,
		runG:   &run.Group{},
		logger: slog.Default(),
		hooks: &hooks{
			onStarts: []HookFunc{},
			onStops:  []HookFunc{},
		},
	}
}
