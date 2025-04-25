package fleet

import (
	"context"
	"github.com/fluxsets/fleet/eventbus"
	"github.com/fluxsets/fleet/option"
	"github.com/oklog/run"
	slogzap "github.com/samber/slog-zap/v2"
	"go.uber.org/zap"
	"gocloud.dev/server/health"
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
	C() Configurer
	O() *option.Option
	Context() context.Context
	ComponentFromProducer(producer ComponentProducer) ([]Component, error)
	Component(components ...Component) error
	Command(cmd CommandFunc) error
	HealthCheck() HealthCheckerRetriever
	Run() error
	EventBus() eventbus.EventBus
	Hooks() Hooks
	Logger(args ...any) *slog.Logger
}

type fleet struct {
	ctx          context.Context
	cancelCtx    context.CancelFunc
	o            option.Option
	runG         *run.Group
	eventBus     eventbus.EventBus
	hooks        *hooks
	logger       *slog.Logger
	c            Configurer
	healthChecks []health.Checker
}

func (ft *fleet) HealthCheck() HealthCheckerRetriever {
	return func() []health.Checker {
		return ft.healthChecks
	}
}

func (ft *fleet) Command(cmd CommandFunc) error {
	return ft.Component(NewCommand(cmd))
}

func (ft *fleet) Close() {
	ft.cancelCtx()
}

func (ft *fleet) Init() error {
	ft.initConfigurer()
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

func (ft *fleet) initConfigurer() {
	configDir := ft.o.ConfigDir
	config := ft.o.Config
	configType := "yaml"
	if ft.o.ConfigType != "" {
		configType = ft.o.ConfigType
	}
	if configDir != "" {

		configDirs := strings.Split(configDir, ",")
		ft.c = newConfigFromDir(configDirs, configType)
	} else if config != "" {
		ft.c = newConfigFromFile(config)
	} else {
		ft.c = newConfigFromDir([]string{"./configs"}, configType)
	}

	ft.c.Merge(ft.o.PropertiesAsMap())
}

func (ft *fleet) ComponentFromProducer(producer ComponentProducer) ([]Component, error) {
	newFn := producer.ComponentFunc
	options := producer.Option()
	options.ensureDefaults()
	var components []Component
	for i := 0; i < options.Instances; i++ {
		comp := newFn()
		components = append(components, comp)
	}
	return components, ft.Component(components...)
}

func (ft *fleet) C() Configurer {
	return ft.c
}

func (ft *fleet) Logger(args ...any) *slog.Logger {
	return ft.logger.With(args...)
}

func (ft *fleet) Context() context.Context {
	return ft.ctx
}

func (ft *fleet) O() *option.Option {
	return &ft.o
}

func (ft *fleet) Component(components ...Component) error {
	for _, comp := range components {
		ctx, cancel := context.WithCancel(context.Background())
		if err := comp.Init(ft); err != nil {
			cancel()
			return err
		}
		ft.runG.Add(func() error {
			return comp.Start(ctx)
		}, func(err error) {
			comp.Stop(ctx)
			cancel()
		})
		if hc, ok := comp.(health.Checker); ok {
			ft.healthChecks = append(ft.healthChecks, hc)
		}
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

func newFleet(o option.Option) Fleet {
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
