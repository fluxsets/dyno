package dyno

import "context"

type SetupFunc func(ctx context.Context, do Dyno) error

type CLI struct {
	setup  SetupFunc
	ctx    context.Context
	cancel context.CancelFunc
	dyno   Dyno
}

func NewCLI(setup SetupFunc, opts ...Option) *CLI {
	o := opts[0]
	do := newDyno(o)
	return &CLI{
		setup: setup,
		dyno:  do,
	}
}

func (cli *CLI) Run() error {
	if err := cli.setup(cli.ctx, cli.dyno); err != nil {
		return err
	}
	return cli.dyno.Run()
}
