package orbit

import (
	"context"
	"os"
)

type SetupFunc func(ctx context.Context, ob Orbit) error

type App struct {
	setup SetupFunc
	orbit Orbit
}

func NewApp(o Option, setup SetupFunc) *App {
	if o.ID == "" {
		o.ID, _ = os.Hostname()
	}
	ob := New(o)
	return &App{
		setup: setup,
		orbit: ob,
	}
}

func (cli *App) Run() error {
	if err := cli.setup(cli.orbit.Context(), cli.orbit); err != nil {
		return err
	}
	return cli.orbit.Run()
}
