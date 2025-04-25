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

func (flt *fleet) DeployCommand(cmd CommandFunc) error {
	return flt.Deploy(NewCommand(cmd))
}

func (flt *fleet) Close() {
	flt.cancelCtx()
}

func (flt *fleet) Init() error {
	flt.initConfig()
	flt.initLogger()
	flt.hooks.OnStop(func(ctx context.Context) error {
		return flt.EventBus().Close(ctx)
	})
	return nil
}

func (flt *fleet) initLogger() {
	level := slog.LevelDebug
	atomicLevel := zap.NewAtomicLevel()

	zapLevel := zap.DebugLevel
	if flt.o.LogLevel != "" {
		_ = level.UnmarshalText([]byte(flt.o.LogLevel))
		_ = zapLevel.UnmarshalText([]byte(flt.o.LogLevel))
	}
	atomicLevel.SetLevel(zapLevel)

	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = atomicLevel
	//zapConfig.EncoderConfig.EncodeTime= zap.En
	zapLogger, _ := zapConfig.Build()
	slog.SetLogLoggerLevel(level)
	logger := slog.New(slogzap.Option{Level: level, Logger: zapLogger}.NewZapHandler())
	logger = logger.With("version", flt.o.Version, "service_name", flt.o.Name, "service_id", flt.o.ID)
	slog.SetDefault(logger)
	flt.logger = logger
}

func (flt *fleet) initConfig() {
	configPaths := strings.Split(flt.o.Conf, ",")

	flt.c = newConfig(configPaths, []string{"yaml"})
	flt.c.Merge(flt.o.KWArgsAsMap())
}

func (flt *fleet) DeployFromProducer(producer DeploymentProducer, options DeploymentOptions) ([]Deployment, error) {
	options.ensureDefaults()
	var deployments []Deployment
	for i := 0; i < options.Instances; i++ {
		dep := producer()
		deployments = append(deployments, dep)
	}
	return deployments, flt.Deploy(deployments...)
}

func (flt *fleet) Config() Config {
	return flt.c
}

func (flt *fleet) Logger(args ...any) *slog.Logger {
	return flt.logger.With(args...)
}

func (flt *fleet) Context() context.Context {
	return flt.ctx
}

func (flt *fleet) Option() option.Option {
	return flt.o
}

func (flt *fleet) Deploy(deployments ...Deployment) error {
	for _, dep := range deployments {
		ctx, cancel := context.WithCancel(context.Background())
		if err := dep.Init(flt); err != nil {
			cancel()
			return err
		}
		flt.runG.Add(func() error {
			return dep.Start(ctx)
		}, func(err error) {
			dep.Stop(ctx)
			cancel()
		})
	}
	return nil
}

func (flt *fleet) Run() error {
	flt.Logger().Info("starting")
	flt.runG.Add(func() error {
		flt.Logger().Info("calling on start hooks")
		for _, fn := range flt.hooks.onStarts {
			if err := fn(flt.ctx); err != nil {
				return err
			}
		}
		select {
		case <-flt.ctx.Done():
			return nil
		}
	}, func(err error) {
		flt.Close()
	})

	flt.runG.Add(func() error {
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-flt.ctx.Done():
			return nil
		case <-exit:
			return nil
		}
	}, func(err error) {
		flt.Logger().Info("shutting down")
		ctx, cancelCtx := context.WithTimeout(context.TODO(), flt.o.ShutdownTimeout)
		defer cancelCtx()
		flt.Logger().Info("calling on stop hooks")
		for _, fn := range flt.hooks.onStops {
			fn := fn
			if err := fn(ctx); err != nil {
				flt.logger.ErrorContext(flt.ctx, "post stop func called error", "error", err)
			}
		}
	})
	return flt.runG.Run()
}

func (flt *fleet) EventBus() eventbus.EventBus {
	return flt.eventBus
}

func (flt *fleet) Hooks() Hooks {
	return flt.hooks
}

func newHyper(o option.Option) Fleet {
	o.EnsureDefaults()
	flt := &fleet{
		o:    o,
		runG: &run.Group{},
		hooks: &hooks{
			onStarts: []HookFunc{},
			onStops:  []HookFunc{},
		},
		eventBus: eventbus.New(),
	}
	flt.ctx, flt.cancelCtx = context.WithCancel(context.Background())
	if err := flt.Init(); err != nil {
		log.Fatal(err)
	}
	return flt
}
