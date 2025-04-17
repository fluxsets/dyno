package orbit

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

type Orbit interface {
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

type orbit struct {
	ctx       context.Context
	cancelCtx context.CancelFunc
	o         Option
	runG      *run.Group
	eventBus  EventBus
	hooks     *hooks
	logger    *slog.Logger
	c         Config
}

func (ob *orbit) Close() {
	ob.cancelCtx()
}

func (ob *orbit) Init() error {
	ob.initConfig()
	ob.initLogger()
	ob.hooks.OnStop(func(ctx context.Context) error {
		return ob.EventBus().Close(ctx)
	})
	return nil
}

func (ob *orbit) initLogger() {
	level := slog.LevelDebug
	atomicLevel := zap.NewAtomicLevel()

	zapLevel := zap.DebugLevel
	if ob.o.LogLevel != "" {
		_ = level.UnmarshalText([]byte(ob.o.LogLevel))
		_ = zapLevel.UnmarshalText([]byte(ob.o.LogLevel))
	}
	atomicLevel.SetLevel(zapLevel)

	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = atomicLevel
	//zapConfig.EncoderConfig.EncodeTime= zap.En
	zapLogger, _ := zapConfig.Build()
	slog.SetLogLoggerLevel(level)
	logger := slog.New(slogzap.Option{Level: level, Logger: zapLogger}.NewZapHandler())
	logger = logger.With("version", ob.o.Version, "service_name", ob.o.Name, "service_id", ob.o.ID)
	slog.SetDefault(logger)
	ob.logger = logger
}

func (ob *orbit) initConfig() {
	configPaths := strings.Split(ob.o.Conf, ",")

	ob.c = newConfig(configPaths, []string{"yaml"})
	ob.c.Merge(ob.o.KWArgsAsMap())
}

func (ob *orbit) DeployFromProducer(producer DeploymentProducer, options DeploymentOptions) ([]Deployment, error) {
	options.ensureDefaults()
	var deployments []Deployment
	for i := 0; i < options.Instances; i++ {
		dep := producer()
		deployments = append(deployments, dep)
	}
	return deployments, ob.Deploy(deployments...)
}

func (ob *orbit) Config() Config {
	return ob.c
}

func (ob *orbit) Logger(args ...any) *slog.Logger {
	return ob.logger.With(args...)
}

func (ob *orbit) Context() context.Context {
	return ob.ctx
}

func (ob *orbit) Option() Option {
	return ob.o
}

func (ob *orbit) Deploy(deployments ...Deployment) error {
	for _, dep := range deployments {
		ctx, cancel := context.WithCancel(context.Background())
		if err := dep.Init(ob); err != nil {
			cancel()
			return err
		}
		ob.runG.Add(func() error {
			return dep.Start(ctx)
		}, func(err error) {
			dep.Stop(ctx)
			cancel()
		})
	}
	return nil
}

func (ob *orbit) Run() error {
	ob.Logger().Info("starting")
	for _, fn := range ob.hooks.onStarts {
		if err := fn(ob.ctx); err != nil {
			return err
		}
	}
	ob.runG.Add(func() error {
		exit := make(chan os.Signal, 1)
		signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
		select {
		case <-ob.ctx.Done():
			return nil
		case <-exit:
			return nil
		}
	}, func(err error) {
		ob.Logger().Info("shutting down")
		ctx, cancelCtx := context.WithTimeout(context.TODO(), ob.o.ShutdownTimeout)
		defer cancelCtx()
		for _, fn := range ob.hooks.onStops {
			fn := fn
			if err := fn(ctx); err != nil {
				ob.logger.ErrorContext(ob.ctx, "post stop func called error", "error", err)
			}
		}
	})
	return ob.runG.Run()
}

func (ob *orbit) EventBus() EventBus {
	return ob.eventBus
}

func (ob *orbit) Hooks() Hooks {
	return ob.hooks
}

func New(o Option) Orbit {
	o.ensureDefaults()
	ob := &orbit{
		o:    o,
		runG: &run.Group{},
		hooks: &hooks{
			onStarts: []HookFunc{},
			onStops:  []HookFunc{},
		},
		eventBus: newEventBus(),
	}
	ob.ctx, ob.cancelCtx = context.WithCancel(context.Background())
	if err := ob.Init(); err != nil {
		log.Fatal(err)
	}
	return ob
}
