package dyno

import (
	"context"
	"os"
)

type SetupFunc func(ctx context.Context, do Dyno) error

type CLI struct {
	setup SetupFunc
	dyno  Dyno
}

func NewCLI(o Option, setup SetupFunc) *CLI {
	if o.ID == "" {
		o.ID, _ = os.Hostname()
	}
	do := New(o)
	return &CLI{
		setup: setup,
		dyno:  do,
	}
}

func (cli *CLI) Run() error {
	if err := cli.setup(cli.dyno.Context(), cli.dyno); err != nil {
		return err
	}
	return cli.dyno.Run()
}
