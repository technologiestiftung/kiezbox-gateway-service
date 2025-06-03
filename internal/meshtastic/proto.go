// Package meshtastic provides utility functions for communication with a meshtastic device over serial
package meshtastic

import (
	"context"
	"kiezbox/internal/github.com/meshtastic/go/generated"
	"log"
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
			log.Println("MessageHandler context canceled, shutting down.")
			return
		case fromRadio, ok := <-mts.FromChan:
			if !ok {
				// Channel closed, exit the handler
				log.Println("FromChan closed, shutting down MessageHandler.")
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
							log.Println("Failed to unmarshal KiezboxMessage: %w", err)
						} else {
							log.Println("Sucessfully extracted KiezboxMessage:")
							debugPrintProtobuf(&KiezboxMessage)
							mts.KBChan <- &KiezboxMessage
						}
					// Extract AdminMessage
					case generated.PortNum_ADMIN_APP:
						var AdminMessage generated.AdminMessage
						err := proto.Unmarshal(v.Decoded.Payload, &AdminMessage)
						if err != nil {
							log.Println("Failed to unmarshal AdminMessage: %w", err)
						} else {
							log.Println("Sucessfully extracted AdminMessage:")
							debugPrintProtobuf(&AdminMessage)
							mts.ConfigChan <- &AdminMessage
						}
					default:
						log.Println("Payload variant not an accepted type of message")
					}
				default:
					// log.Println("Payload variant is encrypted")
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
				// log.Println("Payload variant is not 'packet'")
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
		log.Printf("Failed to marshal Protobuf message to text: %v", err)
		return
	}

	// Print the formatted Protobuf message
	log.Println("Protobuf message content (Text):")
	log.Println(string(textData))
}
