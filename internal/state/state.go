package config

import (
	"kiezbox/internal/github.com/meshtastic/go/generated"
)

type GatewayState struct {
	Mode generated.KiezboxMessage_Mode
}

// Global gateway service config
var State GatewayState
