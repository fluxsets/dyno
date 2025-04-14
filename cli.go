package dyno

import (
	"context"
	"os"
)

type SetupFunc func(ctx context.Context, do Dyno) error

type App struct {
	setup SetupFunc
	dyno  Dyno
}

func NewApp(o Option, setup SetupFunc) *App {
	if o.ID == "" {
		o.ID, _ = os.Hostname()
	}
	do := New(o)
	return &App{
		setup: setup,
		dyno:  do,
	}
}

func (cli *App) Run() error {
	if err := cli.setup(cli.dyno.Context(), cli.dyno); err != nil {
		return err
	}
	return cli.dyno.Run()
}
