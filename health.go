package dyno

import (
	"errors"
	"gocloud.dev/server/health"
	"sync"
)

type HealthCheck struct {
	healthy bool
	mu      sync.RWMutex
}

func (check *HealthCheck) SetHealthy(healthy bool) {
	check.mu.Lock()
	defer check.mu.Unlock()
	check.healthy = healthy
}

func (check *HealthCheck) CheckHealth() error {
	check.mu.RLock()
	defer check.mu.RUnlock()
	if !check.healthy {
		return errors.New("unhealthy")
	}
	return nil
}

var _ health.Checker = new(HealthCheck)
