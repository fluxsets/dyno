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

func New(o Option, setup SetupFunc) *App {
	if o.ID == "" {
		o.ID, _ = os.Hostname()
	}
	ob := newOrbit(o)
	return &App{
		setup: setup,
		orbit: ob,
	}
}

func (app *App) Run() {
	if err := app.RunE(); err != nil {
		panic(err)
	}
}

func (app *App) RunE() error {
	if err := app.setup(app.orbit.Context(), app.orbit); err != nil {
		return err
	}
	return app.orbit.Run()
}
