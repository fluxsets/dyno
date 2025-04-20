package hyper

import (
	"context"
	"github.com/fluxsets/hyper/option"
	"os"
)

type SetupFunc func(ctx context.Context, hyp Hyper) error

type App struct {
	setup SetupFunc
	hyper Hyper
}

func New(o option.Option, setup SetupFunc) *App {
	if o.ID == "" {
		o.ID, _ = os.Hostname()
	}
	hyp := newHyper(o)
	return &App{
		setup: setup,
		hyper: hyp,
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
