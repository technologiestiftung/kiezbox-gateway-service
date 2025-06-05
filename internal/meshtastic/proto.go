// Package meshtastic provides utility functions for communication with a meshtastic device over serial
package meshtastic

import (
	"context"
	"kiezbox/internal/github.com/meshtastic/go/generated"
	"log/slog"
	"sync"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// MessageHandler takes a FromRadio protobuf KiezboxMessage or AdminMessage and extracts it if possible
// It returns the containing message or nil otherwise
func (mts *MTSerial) MessageHandler(ctx context.Context, wg *sync.WaitGroup) {
	// Decrement WaitGroup when function exits
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			// Exit gracefully when the context is canceled
			slog.Info("MessageHandler context canceled, shutting down.")
			return
		case fromRadio, ok := <-mts.FromChan:
			if !ok {
				// Channel closed, exit the handler
				slog.Info("FromChan closed, shutting down MessageHandler.")
				return
			}
			// debugPrintProtobuf(fromRadio)
			switch v := fromRadio.PayloadVariant.(type) {
			case *generated.FromRadio_Packet:
				switch v := v.Packet.PayloadVariant.(type) {
				case *generated.MeshPacket_Decoded:
					// Extract the message according to its type
					switch v.Decoded.Portnum {
					// Extract KiezboxMessage
					case generated.PortNum_KIEZBOX_CONTROL_APP:
						var KiezboxMessage generated.KiezboxMessage
						err := proto.Unmarshal(v.Decoded.Payload, &KiezboxMessage)
						if err != nil {
							slog.Error("Failed to unmarshal KiezboxMessage", "err", err)
						} else {
							slog.Info("Sucessfully extracted KiezboxMessage")
							debugPrintProtobuf(&KiezboxMessage)
							mts.KBChan <- &KiezboxMessage
						}
					// Extract AdminMessage
					case generated.PortNum_ADMIN_APP:
						var AdminMessage generated.AdminMessage
						err := proto.Unmarshal(v.Decoded.Payload, &AdminMessage)
						if err != nil {
							slog.Error("Failed to unmarshal AdminMessage", "err", err)
						} else {
							slog.Info("Sucessfully extracted AdminMessage")
							debugPrintProtobuf(&AdminMessage)
							mts.ConfigChan <- &AdminMessage
						}
					default:
						slog.Info("Payload variant not an accepted type of message")
					}
				default:
					// slog.Info("Payload variant is encrypted")
				}
			case *generated.FromRadio_MyInfo:
				{
					mts.MyInfo = v.MyInfo
					mts.WaitInfo.Done()
				}
			//Device rebooted, so we ask for config again to initialize communication
			case *generated.FromRadio_Rebooted:
				{
					mts.WaitInfo.Add(1)
					mts.WantConfig()
				}
			default:
				// slog.Info("Payload variant is not 'packet'")
			}
		}
	}
}

// debugPrintProtobuf takes a protobuf message and prints it in a pretty way for debugging
func debugPrintProtobuf(message proto.Message) {
	// Convert the Protobuf message to text format
	textData, err := prototext.MarshalOptions{
		Multiline: true, // Use multiline output for readability
	}.Marshal(message)

	if err != nil {
		slog.Error("Failed to marshal Protobuf message to text", "err", err)
		return
	}

	// Print the formatted Protobuf message
	slog.Info("Protobuf message content (Text):")
	slog.Info(string(textData))
}
