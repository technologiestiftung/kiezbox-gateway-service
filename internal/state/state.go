package state

import (
	"kiezbox/internal/github.com/meshtastic/go/generated"
	"sync"
)

type GatewayState struct {
	mutex sync.RWMutex
	mode  generated.KiezboxMessage_Mode
}

// Global gateway service config
var State GatewayState

// SetMode safely sets the Mode field of the global State
func SetMode(newMode generated.KiezboxMessage_Mode) {
	State.mutex.Lock()
	defer State.mutex.Unlock()
	State.mode = newMode
}

// GetMode safely gets the Mode field of the global State
func GetMode() generated.KiezboxMessage_Mode {
	State.mutex.RLock()
	defer State.mutex.RUnlock()
	return State.mode
}
