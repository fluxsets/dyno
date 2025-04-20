package hyper

import (
	"context"
	"github.com/fluxsets/hyper/eventbus"
	"github.com/fluxsets/hyper/option"
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

type hyper struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	o         option.Option
	runG      *run.Group
	eventBus  eventbus.EventBus
	hooks     *hooks
	logger    *slog.Logger
	c         Config
}

func (hyp *hyper) DeployCommand(cmd CommandFunc) error {
	return hyp.Deploy(NewCommand(cmd))
}

func (hyp *hyper) Close() {
	hyp.cancelCtx()
}

func (hyp *hyper) Init() error {
	hyp.initConfig()
	hyp.initLogger()
	hyp.hooks.OnStop(func(ctx context.Context) error {
		return hyp.EventBus().Close(ctx)
	})
	return nil
}

func (hyp *hyper) initLogger() {
	level := slog.LevelDebug
	atomicLevel := zap.NewAtomicLevel()

	zapLevel := zap.DebugLevel
	if hyp.o.LogLevel != "" {
		_ = level.UnmarshalText([]byte(hyp.o.LogLevel))
		_ = zapLevel.UnmarshalText([]byte(hyp.o.LogLevel))
	}
	atomicLevel.SetLevel(zapLevel)

	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = atomicLevel
	//zapConfig.EncoderConfig.EncodeTime= zap.En
	zapLogger, _ := zapConfig.Build()
	slog.SetLogLoggerLevel(level)
	logger := slog.New(slogzap.Option{Level: level, Logger: zapLogger}.NewZapHandler())
	logger = logger.With("version", hyp.o.Version, "service_name", hyp.o.Name, "service_id", hyp.o.ID)
	slog.SetDefault(logger)
	hyp.logger = logger
}

func (hyp *hyper) initConfig() {
	configPaths := strings.Split(hyp.o.Conf, ",")

	hyp.c = newConfig(configPaths, []string{"yaml"})
	hyp.c.Merge(hyp.o.KWArgsAsMap())
}

func (hyp *hyper) DeployFromProducer(producer DeploymentProducer, options DeploymentOptions) ([]Deployment, error) {
	options.ensureDefaults()
	var deployments []Deployment
	for i := 0; i < options.Instances; i++ {
		dep := producer()
		deployments = append(deployments, dep)
	}
	return deployments, hyp.Deploy(deployments...)
}

func (hyp *hyper) Config() Config {
	return hyp.c
}

func (hyp *hyper) Logger(args ...any) *slog.Logger {
	return hyp.logger.With(args...)
}

func (hyp *hyper) Context() context.Context {
	return hyp.ctx
}

func (hyp *hyper) Option() option.Option {
	return hyp.o
}

func (hyp *hyper) Deploy(deployments ...Deployment) error {
	for _, dep := range deployments {
		ctx, cancel := context.WithCancel(context.Background())
		if err := dep.Init(hyp); err != nil {
			cancel()
			return err
		}
		hyp.runG.Add(func() error {
			return dep.Start(ctx)
		}, func(err error) {
			dep.Stop(ctx)
			cancel()
		})
	}
	return nil
}

func (hyp *hyper) Run() error {
	hyp.Logger().Info("starting")
	hyp.runG.Add(func() error {
		hyp.Logger().Info("calling on start hooks")
		for _, fn := range hyp.hooks.onStarts {
			if err := fn(hyp.ctx); err != nil {
				return err
			}
		}
		select {
		case <-hyp.ctx.Done():
			return nil
		}
	}, func(err error) {
		hyp.Close()
	})

	hyp.runG.Add(func() error {
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-hyp.ctx.Done():
			return nil
		case <-exit:
			return nil
		}
	}, func(err error) {
		hyp.Logger().Info("shutting down")
		ctx, cancelCtx := context.WithTimeout(context.TODO(), hyp.o.ShutdownTimeout)
		defer cancelCtx()
		hyp.Logger().Info("calling on stop hooks")
		for _, fn := range hyp.hooks.onStops {
			fn := fn
			if err := fn(ctx); err != nil {
				hyp.logger.ErrorContext(hyp.ctx, "post stop func called error", "error", err)
			}
		}
	})
	return hyp.runG.Run()
}

func (hyp *hyper) EventBus() eventbus.EventBus {
	return hyp.eventBus
}

func (hyp *hyper) Hooks() Hooks {
	return hyp.hooks
}

func newHyper(o option.Option) Hyper {
	o.EnsureDefaults()
	hyp := &hyper{
		o:    o,
		runG: &run.Group{},
		hooks: &hooks{
			onStarts: []HookFunc{},
			onStops:  []HookFunc{},
		},
		eventBus: eventbus.New(),
	}
	hyp.ctx, hyp.cancelCtx = context.WithCancel(context.Background())
	if err := hyp.Init(); err != nil {
		log.Fatal(err)
	}
	return hyp
}
