package marshal

import (
	"time"

	"kiezbox/internal/github.com/meshtastic/go/generated"
)

// CreateKiezboxMessage is a helper function that generates a basic KiezboxMessage
func CreateKiezboxMessage() *generated.KiezboxMessage {
	return &generated.KiezboxMessage{
		Update: &generated.KiezboxMessage_Update{
			Meta: &generated.KiezboxMessage_Meta{
				BoxId:  1, // Example value
				DistId: 2, // Example value
			},
			UnixTime: time.Now().Unix(), // Current Unix timestamp
			Core: &generated.KiezboxMessage_Core{
				Mode: generated.KiezboxMessage_normal,
				Router: &generated.KiezboxMessage_Router{
					Powered: true,
				},
				Values: &generated.KiezboxMessage_CoreValues{},
			},
		},
	}
}
