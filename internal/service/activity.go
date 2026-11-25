package service

import "sync"

var (
	activityMu        sync.Mutex
	activeInteractive int64
	maintenanceActive bool
)

// BeginInteractiveSession registers an active interactive WebSocket session
// (install, pair, login, or scan). It refuses to start while a maintenance
// operation such as a frontend hot-update switch is in progress.
func BeginInteractiveSession() (func(), bool) {
	activityMu.Lock()
	if maintenanceActive {
		activityMu.Unlock()
		return nil, false
	}
	activeInteractive++
	activityMu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			activityMu.Lock()
			if activeInteractive > 0 {
				activeInteractive--
			}
			activityMu.Unlock()
		})
	}, true
}

// TryReserveMaintenance acquires exclusive maintenance access while no
// interactive WebSocket session is running.
func TryReserveMaintenance() (func(), bool) {
	activityMu.Lock()
	if maintenanceActive || activeInteractive > 0 {
		activityMu.Unlock()
		return nil, false
	}
	maintenanceActive = true
	activityMu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			activityMu.Lock()
			maintenanceActive = false
			activityMu.Unlock()
		})
	}, true
}

// ActiveInteractiveSessions returns the count of active interactive sessions.
func ActiveInteractiveSessions() int64 {
	activityMu.Lock()
	defer activityMu.Unlock()
	return activeInteractive
}
