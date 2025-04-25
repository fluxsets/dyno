package boot

import (
	"github.com/fluxsets/fleet"
	"github.com/fluxsets/fleet/option"
	"log"
)

type Bootstrap struct {
	StartHooks []fleet.HookFunc
	StopHooks  []fleet.HookFunc
	Components []fleet.Component
	o          *option.Option
	c          fleet.Configurer
}

func NewBootstrap(o *option.Option, c fleet.Configurer) *Bootstrap {
	return &Bootstrap{
		StartHooks: []fleet.HookFunc{},
		StopHooks:  []fleet.HookFunc{},
		Components: []fleet.Component{},
		o:          o,
		c:          c,
	}
}

func (b *Bootstrap) Wire(fl fleet.Fleet) {
	fl.Hooks().OnStart(b.StopHooks...)
	fl.Hooks().OnStop(b.StopHooks...)
	if err := fl.Component(b.Components...); err != nil {
		log.Fatal(err)
	}
}
