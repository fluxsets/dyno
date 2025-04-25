package fleet

import (
	"context"
	"github.com/fluxsets/fleet/eventbus"
	"github.com/fluxsets/fleet/option"
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

type Fleet interface {
	Init() error
	Close()
	Config() Config
	Option() option.Option
	Context() context.Context
	DeployFromProducer(producer DeploymentProducer, options DeploymentOptions) ([]Deployment, error)
	Deploy(deployments ...Deployment) error
	DeployCommand(cmd CommandFunc) error
	Run() error
	EventBus() eventbus.EventBus
	Hooks() Hooks
	Logger(args ...any) *slog.Logger
}

type fleet struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	o         option.Option
	runG      *run.Group
	eventBus  eventbus.EventBus
	hooks     *hooks
	logger    *slog.Logger
	c         Config
}

func (ft *fleet) DeployCommand(cmd CommandFunc) error {
	return ft.Deploy(NewCommand(cmd))
}

func (ft *fleet) Close() {
	ft.cancelCtx()
}

func (ft *fleet) Init() error {
	ft.initConfig()
	ft.initLogger()
	ft.hooks.OnStop(func(ctx context.Context) error {
		return ft.EventBus().Close(ctx)
	})
	return nil
}

func (ft *fleet) initLogger() {
	level := slog.LevelDebug
	atomicLevel := zap.NewAtomicLevel()

	zapLevel := zap.DebugLevel
	if ft.o.LogLevel != "" {
		_ = level.UnmarshalText([]byte(ft.o.LogLevel))
		_ = zapLevel.UnmarshalText([]byte(ft.o.LogLevel))
	}
	atomicLevel.SetLevel(zapLevel)

	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = atomicLevel
	//zapConfig.EncoderConfig.EncodeTime= zap.En
	zapLogger, _ := zapConfig.Build()
	slog.SetLogLoggerLevel(level)
	logger := slog.New(slogzap.Option{Level: level, Logger: zapLogger}.NewZapHandler())
	logger = logger.With("version", ft.o.Version, "service_name", ft.o.Name, "service_id", ft.o.ID)
	slog.SetDefault(logger)
	ft.logger = logger
}

func (ft *fleet) initConfig() {
	configPaths := strings.Split(ft.o.Conf, ",")

	ft.c = newConfig(configPaths, []string{"yaml"})
	ft.c.Merge(ft.o.KWArgsAsMap())
}

func (ft *fleet) DeployFromProducer(producer DeploymentProducer, options DeploymentOptions) ([]Deployment, error) {
	options.ensureDefaults()
	var deployments []Deployment
	for i := 0; i < options.Instances; i++ {
		dep := producer()
		deployments = append(deployments, dep)
	}
	return deployments, ft.Deploy(deployments...)
}

func (ft *fleet) Config() Config {
	return ft.c
}

func (ft *fleet) Logger(args ...any) *slog.Logger {
	return ft.logger.With(args...)
}

func (ft *fleet) Context() context.Context {
	return ft.ctx
}

func (ft *fleet) Option() option.Option {
	return ft.o
}

func (ft *fleet) Deploy(deployments ...Deployment) error {
	for _, dep := range deployments {
		ctx, cancel := context.WithCancel(context.Background())
		if err := dep.Init(ft); err != nil {
			cancel()
			return err
		}
		ft.runG.Add(func() error {
			return dep.Start(ctx)
		}, func(err error) {
			dep.Stop(ctx)
			cancel()
		})
	}
	return nil
}

func (ft *fleet) Run() error {
	ft.Logger().Info("starting")
	ft.runG.Add(func() error {
		ft.Logger().Info("calling on start hooks")
		for _, fn := range ft.hooks.onStarts {
			if err := fn(ft.ctx); err != nil {
				return err
			}
		}
		select {
		case <-ft.ctx.Done():
			return nil
		}
	}, func(err error) {
		ft.Close()
	})

	ft.runG.Add(func() error {
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-ft.ctx.Done():
			return nil
		case <-exit:
			return nil
		}
	}, func(err error) {
		ft.Logger().Info("shutting down")
		ctx, cancelCtx := context.WithTimeout(context.TODO(), ft.o.ShutdownTimeout)
		defer cancelCtx()
		ft.Logger().Info("calling on stop hooks")
		for _, fn := range ft.hooks.onStops {
			fn := fn
			if err := fn(ctx); err != nil {
				ft.logger.ErrorContext(ft.ctx, "post stop func called error", "error", err)
			}
		}
	})
	return ft.runG.Run()
}

func (ft *fleet) EventBus() eventbus.EventBus {
	return ft.eventBus
}

func (ft *fleet) Hooks() Hooks {
	return ft.hooks
}

func newHyper(o option.Option) Fleet {
	o.EnsureDefaults()
	ft := &fleet{
		o:    o,
		runG: &run.Group{},
		hooks: &hooks{
			onStarts: []HookFunc{},
			onStops:  []HookFunc{},
		},
		eventBus: eventbus.New(),
	}
	ft.ctx, ft.cancelCtx = context.WithCancel(context.Background())
	if err := ft.Init(); err != nil {
		log.Fatal(err)
	}
	return ft
}
