package boot

import (
	"github.com/fluxsets/fleet"
)

type Bootstrap struct {
	StartHooks         []fleet.HookFunc
	StopHooks          []fleet.HookFunc
	Components         []fleet.Component
	ComponentProducers []fleet.ComponentProducer
}

func NewBootstrap(
	onStars fleet.OnStartHooks,
	onStops fleet.OnStopHooks,
	components []fleet.Component,
	componentProducers []fleet.ComponentProducer,
) *Bootstrap {
	return &Bootstrap{
		StartHooks:         onStars,
		StopHooks:          onStops,
		Components:         components,
		ComponentProducers: componentProducers,
	}
}

func (b *Bootstrap) Bind(fl fleet.Fleet) error {
	fl.Hooks().OnStart(b.StopHooks...)
	fl.Hooks().OnStop(b.StopHooks...)
	if err := fl.Mount(b.Components...); err != nil {
		return err
	}

	if err := fl.MountFromProducer(b.ComponentProducers...); err != nil {
		return err
	}
	return nil
}
