package hyper

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

type Hyper interface {
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

type hyper struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	o         Option
	runG      *run.Group
	eventBus  EventBus
	hooks     *hooks
	logger    *slog.Logger
	c         Config
}

func (hp *hyper) Close() {
	hp.cancelCtx()
}

func (hp *hyper) Init() error {
	hp.initConfig()
	hp.initLogger()
	hp.hooks.OnStop(func(ctx context.Context) error {
		return hp.EventBus().Close(ctx)
	})
	return nil
}

func (hp *hyper) initLogger() {
	level := slog.LevelDebug
	atomicLevel := zap.NewAtomicLevel()

	zapLevel := zap.DebugLevel
	if hp.o.LogLevel != "" {
		_ = level.UnmarshalText([]byte(hp.o.LogLevel))
		_ = zapLevel.UnmarshalText([]byte(hp.o.LogLevel))
	}
	atomicLevel.SetLevel(zapLevel)

	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = atomicLevel
	//zapConfig.EncoderConfig.EncodeTime= zap.En
	zapLogger, _ := zapConfig.Build()
	slog.SetLogLoggerLevel(level)
	logger := slog.New(slogzap.Option{Level: level, Logger: zapLogger}.NewZapHandler())
	logger = logger.With("version", hp.o.Version, "service_name", hp.o.Name, "service_id", hp.o.ID)
	slog.SetDefault(logger)
	hp.logger = logger
}

func (hp *hyper) initConfig() {
	configPaths := strings.Split(hp.o.Conf, ",")

	hp.c = newConfig(configPaths, []string{"yaml"})
	hp.c.Merge(hp.o.KWArgsAsMap())
}

func (hp *hyper) DeployFromProducer(producer DeploymentProducer, options DeploymentOptions) ([]Deployment, error) {
	options.ensureDefaults()
	var deployments []Deployment
	for i := 0; i < options.Instances; i++ {
		dep := producer()
		deployments = append(deployments, dep)
	}
	return deployments, hp.Deploy(deployments...)
}

func (hp *hyper) Config() Config {
	return hp.c
}

func (hp *hyper) Logger(args ...any) *slog.Logger {
	return hp.logger.With(args...)
}

func (hp *hyper) Context() context.Context {
	return hp.ctx
}

func (hp *hyper) Option() Option {
	return hp.o
}

func (hp *hyper) Deploy(deployments ...Deployment) error {
	for _, dep := range deployments {
		ctx, cancel := context.WithCancel(context.Background())
		if err := dep.Init(hp); err != nil {
			cancel()
			return err
		}
		hp.runG.Add(func() error {
			return dep.Start(ctx)
		}, func(err error) {
			dep.Stop(ctx)
			cancel()
		})
	}
	return nil
}

func (hp *hyper) Run() error {
	hp.Logger().Info("starting")
	for _, fn := range hp.hooks.onStarts {
		if err := fn(hp.ctx); err != nil {
			return err
		}
	}
	hp.runG.Add(func() error {
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-hp.ctx.Done():
			return nil
		case <-exit:
			return nil
		}
	}, func(err error) {
		hp.Logger().Info("shutting down")
		ctx, cancelCtx := context.WithTimeout(context.TODO(), hp.o.ShutdownTimeout)
		defer cancelCtx()
		for _, fn := range hp.hooks.onStops {
			fn := fn
			if err := fn(ctx); err != nil {
				hp.logger.ErrorContext(hp.ctx, "post stop func called error", "error", err)
			}
		}
	})
	return hp.runG.Run()
}

func (hp *hyper) EventBus() EventBus {
	return hp.eventBus
}

func (hp *hyper) Hooks() Hooks {
	return hp.hooks
}

func newHalo(o Option) Hyper {
	o.ensureDefaults()
	hp := &hyper{
		o:    o,
		runG: &run.Group{},
		hooks: &hooks{
			onStarts: []HookFunc{},
			onStops:  []HookFunc{},
		},
		eventBus: newEventBus(),
	}
	hp.ctx, hp.cancelCtx = context.WithCancel(context.Background())
	if err := hp.Init(); err != nil {
		log.Fatal(err)
	}
	return hp
}
