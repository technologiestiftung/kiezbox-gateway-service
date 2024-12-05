package marshal

import (
	"time"

	"kiezbox/internal/github.com/meshtastic/go/generated"
)

// Create enerates a basic KiezboxMessage
func CreateKiezboxMessage() *generated.KiezboxMessage {
	return &generated.KiezboxMessage{
		Update: &generated.KiezboxMessage_Update{
			Meta: &generated.KiezboxMessage_Meta{
				BoxId:  1,
				DistId: 2,
			},
			UnixTime: time.Now().Unix(),
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
