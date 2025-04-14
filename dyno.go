package dyno

import (
	"context"
	"github.com/oklog/run"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

type Dyno interface {
	Init() error
	Config() Config
	Option() Option
	Context() context.Context
	DeployFromProducer(producer DeploymentProducer, options DeploymentOptions) error
	Deploy(deployments ...Deployment) error
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
	c        Config
}

func (do *dyno) Init() error {
	do.initConfig()
	do.initLogger()
	return nil
}

func (do *dyno) initLogger() {
	do.logger = slog.Default().With("logger", "dyno", "version", do.o.Version, "service_name", do.o.Name, "service_id", do.o.ID)
}

func (do *dyno) initConfig() {
	configPaths := strings.Split(do.o.Conf, ",")
	kwargs := map[string]any{}
	args := strings.Split(do.o.KWArgs, ",")
	for _, s := range args {
		kv := strings.Split(s, "=")
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		kwargs[key] = value
	}
	do.c = newConfig(configPaths, []string{"yaml"})
	do.c.Merge(kwargs)
}

func (do *dyno) DeployFromProducer(producer DeploymentProducer, options DeploymentOptions) error {
	var deployments []Deployment
	for i := 0; i < options.Instances; i++ {
		dep := producer()
		deployments = append(deployments, dep)
	}
	return do.Deploy(deployments...)
}

func (do *dyno) Config() Config {
	return do.c
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

func (do *dyno) Deploy(deployments ...Deployment) error {
	for _, dep := range deployments {
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
	do.Logger().Info("starting")
	for _, fn := range do.hooks.onStarts {
		if err := fn(do.ctx); err != nil {
			return err
		}
	}
	do.runG.Add(func() error {
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
		<-exit
		return nil
	}, func(err error) {
		do.Logger().Info("shutting down")
		for _, fn := range do.hooks.onStops {
			fn := fn
			if err := fn(do.ctx); err != nil {
				do.logger.ErrorContext(do.ctx, "post stop func called error", "error", err)
			}
		}
	})
	return do.runG.Run()
}

func (do *dyno) EventBus() EventBus {
	return do.eventBus
}

func (do *dyno) Hooks() Hooks {
	return do.hooks
}

func New(o Option) Dyno {
	do := &dyno{
		o:    o,
		runG: &run.Group{},
		hooks: &hooks{
			onStarts: []HookFunc{},
			onStops:  []HookFunc{},
		},
	}

	if err := do.Init(); err != nil {
		log.Fatal(err)
	}
	return do
}
