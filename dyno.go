package dyno

import (
	"context"
	"github.com/oklog/run"
	slogzap "github.com/samber/slog-zap/v2"
	"go.uber.org/zap"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

type Dyno interface {
	Init() error
	Close()
	Config() Config
	Option() Option
	Context() context.Context
	DeployFromProducer(producer DeploymentProducer, options DeploymentOptions) ([]Deployment, error)
	Deploy(deployments ...Deployment) error
	Run() error
	EventBus() EventBus
	Hooks() Hooks
	Logger(args ...any) *slog.Logger
}

type dyno struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	o         Option
	runG      *run.Group
	eventBus  EventBus
	hooks     *hooks
	logger    *slog.Logger
	c         Config
}

func (do *dyno) Close() {
	do.cancelCtx()
}

func (do *dyno) Init() error {
	do.initConfig()
	do.initLogger()
	do.hooks.OnStop(func(ctx context.Context) error {
		return do.EventBus().Close(ctx)
	})
	return nil
}

func (do *dyno) initLogger() {
	level := slog.LevelDebug
	if do.o.LogLevel != "" {
		_ = level.UnmarshalText([]byte(do.o.LogLevel))
	}
	zapLogger, _ := zap.NewProduction()
	logger := slog.New(slogzap.Option{Level: level, Logger: zapLogger}.NewZapHandler())
	logger = logger.With("logger", "dyno", "version", do.o.Version, "service_name", do.o.Name, "service_id", do.o.ID)
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(level)
	do.logger = logger
}

func (do *dyno) initConfig() {
	configPaths := strings.Split(do.o.Conf, ",")

	do.c = newConfig(configPaths, []string{"yaml"})
	do.c.Merge(do.o.KWArgsAsMap())
}

func (do *dyno) DeployFromProducer(producer DeploymentProducer, options DeploymentOptions) ([]Deployment, error) {
	options.ensureDefaults()
	var deployments []Deployment
	for i := 0; i < options.Instances; i++ {
		dep := producer()
		deployments = append(deployments, dep)
	}
	return deployments, do.Deploy(deployments...)
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
		select {
		case <-do.ctx.Done():
			return nil
		case <-exit:
			return nil
		}
	}, func(err error) {
		do.Logger().Info("shutting down")
		ctx, cancelCtx := context.WithTimeout(context.TODO(), do.o.ShutdownTimeout)
		defer cancelCtx()
		for _, fn := range do.hooks.onStops {
			fn := fn
			if err := fn(ctx); err != nil {
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
	o.ensureDefaults()
	do := &dyno{
		o:    o,
		runG: &run.Group{},
		hooks: &hooks{
			onStarts: []HookFunc{},
			onStops:  []HookFunc{},
		},
		eventBus: newEventBus(),
	}
	do.ctx, do.cancelCtx = context.WithCancel(context.Background())
	if err := do.Init(); err != nil {
		log.Fatal(err)
	}
	return do
}
