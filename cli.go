package fleet

import (
	"context"
	"github.com/fluxsets/fleet/option"
	"os"
)

type SetupFunc func(ctx context.Context, ft Fleet) error

type App struct {
	setup SetupFunc
	fleet Fleet
}

func New(o option.Option, setup SetupFunc) *App {
	if o.ID == "" {
		o.ID, _ = os.Hostname()
	}
	ft := newHyper(o)
	return &App{
		setup: setup,
		fleet: ft,
	}
}

func (app *App) Run() {
	if err := app.RunE(); err != nil {
		panic(err)
	}
}

func (app *App) RunE() error {
	if err := app.setup(app.fleet.Context(), app.fleet); err != nil {
		return err
	}
	return app.fleet.Run()
}
