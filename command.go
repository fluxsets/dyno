package orbit

import "context"

type CommandFunc func(ctx context.Context) error

func NewCommand(fn CommandFunc) Deployment {
	return &command{fn: fn}
}

type command struct {
	fn    CommandFunc
	orbit Orbit
}

func (cmd *command) CheckHealth() error {
	return nil
}

func (cmd *command) Name() string {
	return "command"
}

func (cmd *command) Init(ob Orbit) error {
	cmd.orbit = ob
	return nil
}

func (cmd *command) Start(ctx context.Context) error {
	return cmd.fn(ctx)
}

func (cmd *command) Stop(ctx context.Context) {
	cmd.orbit.Close()
}
