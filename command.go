package dyno

import "context"

type CommandFunc func(ctx context.Context) error

func NewCommand(fn CommandFunc) Deployment {
	return &command{fn: fn}
}

type command struct {
	fn   CommandFunc
	dyno Dyno
}

func (cmd *command) Name() string {
	return "command"
}

func (cmd *command) Init(do Dyno) error {
	cmd.dyno = do
	return nil
}

func (cmd *command) Start(ctx context.Context) error {
	return cmd.fn(ctx)
}

func (cmd *command) Stop(ctx context.Context) {
	cmd.dyno.Close()
}
