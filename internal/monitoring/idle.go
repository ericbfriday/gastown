package monitoring

import (
	"context"
	"time"
)

// IdleDetector monitors agents and marks them as idle after inactivity.
type IdleDetector struct {
	tracker *StatusTracker
	config  IdleConfig
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// NewIdleDetector creates a new idle detector.
func NewIdleDetector(tracker *StatusTracker, config IdleConfig) *IdleDetector {
	return &IdleDetector{
		tracker: tracker,
		config:  config,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
}

// Start begins idle detection monitoring.
// Runs in a goroutine, checking for idle agents at the configured interval.
func (d *IdleDetector) Start(ctx context.Context) {
	if !d.config.Enabled {
		close(d.doneCh)
		return
	}

	go d.run(ctx)
}

// run is the main idle detection loop.
func (d *IdleDetector) run(ctx context.Context) {
	defer close(d.doneCh)

	ticker := time.NewTicker(d.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-d.stopCh:
			return
		case <-ticker.C:
			d.checkIdle()
		}
	}
}

// checkIdle performs idle check on all tracked agents.
func (d *IdleDetector) checkIdle() {
	idleAgents := d.tracker.CheckIdle()
	if len(idleAgents) > 0 {
		// Could emit events here if needed
		// For now, just let the tracker handle the state update
	}
}

// Stop stops the idle detector.
func (d *IdleDetector) Stop() {
	close(d.stopCh)
	<-d.doneCh // Wait for goroutine to finish
}

// UpdateConfig updates the idle detection configuration.
func (d *IdleDetector) UpdateConfig(config IdleConfig) {
	d.config = config
	d.tracker.SetIdleConfig(config)
}

// GetConfig returns the current idle detection configuration.
func (d *IdleDetector) GetConfig() IdleConfig {
	return d.config
}
