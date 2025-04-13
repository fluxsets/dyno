package dyno

import "context"

type Hooks interface {
	PreStart(fns ...HookFunc)
	PostStop(fns ...HookFunc)
}

type HookFunc func(ctx context.Context) error

type hooks struct {
	preStartFuncs []HookFunc
	postStopFuncs []HookFunc
}

func (hooks *hooks) PreStart(fns ...HookFunc) {
	hooks.preStartFuncs = append(hooks.preStartFuncs, fns...)
}

func (hooks *hooks) PostStop(fns ...HookFunc) {
	hooks.postStopFuncs = append(hooks.postStopFuncs, fns...)
}

var _ Hooks = new(hooks)
