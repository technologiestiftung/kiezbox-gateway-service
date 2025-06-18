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

// GetMode safely gets the Mode field of the global State as a string
func GetMode() string {
	State.mutex.RLock()
	defer State.mutex.RUnlock()
	modeInt := int32(State.mode)
	modeStr, ok := generated.KiezboxMessage_Mode_name[modeInt]
	if ok {
		return modeStr
	}
	return ""
}
