package boot

import (
	"github.com/fluxsets/fleet"
	"log"
)

type Bootstrap struct {
	StartHooks []fleet.HookFunc
	StopHooks  []fleet.HookFunc
	Components []fleet.Component
}

func NewBootstrap() *Bootstrap {
	return &Bootstrap{
		StartHooks: []fleet.HookFunc{},
		StopHooks:  []fleet.HookFunc{},
		Components: []fleet.Component{},
	}
}

func (b *Bootstrap) Wire(fl fleet.Fleet) {
	fl.Hooks().OnStart(b.StopHooks...)
	fl.Hooks().OnStop(b.StopHooks...)
	if err := fl.Component(b.Components...); err != nil {
		log.Fatal(err)
	}
}
