package hyper

import (
	"context"
	"os"
)

type SetupFunc func(ctx context.Context, hp Hyper) error

type App struct {
	setup SetupFunc
	hyper Hyper
}

func New(o Option, setup SetupFunc) *App {
	if o.ID == "" {
		o.ID, _ = os.Hostname()
	}
	hp := newHalo(o)
	return &App{
		setup: setup,
		hyper: hp,
	}
}

func (app *App) Run() {
	if err := app.RunE(); err != nil {
		panic(err)
	}
}

func (app *App) RunE() error {
	if err := app.setup(app.hyper.Context(), app.hyper); err != nil {
		return err
	}
	return app.hyper.Run()
}
