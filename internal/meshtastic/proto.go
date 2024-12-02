// Package meshtastic provides utility functions for communication with a meshtastic device over serial
package meshtastic

import (
	"fmt"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"kiezbox/internal/github.com/meshtastic/go/generated"
)

// ExtractKBMessage takes a FromRadio protobuf message and extracts a KiezboxMessage if possible
// It returns the containing KiezboxMessage or nil otherwise
func ExtractKBMessage(fromRadio *generated.FromRadio) *generated.KiezboxMessage {
	debugPrintProtobuf(fromRadio)
	switch v := fromRadio.PayloadVariant.(type) {
	case *generated.FromRadio_Packet:
		switch v := v.Packet.PayloadVariant.(type) {
		case *generated.MeshPacket_Decoded:
			switch v.Decoded.Portnum {
			case generated.PortNum_KIEZBOX_CONTROL_APP:
				var KiezboxMessage generated.KiezboxMessage
				err := proto.Unmarshal(v.Decoded.Payload, &KiezboxMessage)
				if err != nil {
					fmt.Println("failed to unmarshal KiezboxMessage: %w", err)
				} else {
					fmt.Println("Extracted KiezboxMessage:")
					debugPrintProtobuf(&KiezboxMessage)
					return &KiezboxMessage
				}
			default:
				fmt.Println("Payload variant not a Kiezbox Message")
			}
		default:
			fmt.Println("Payload variant is encrypted")
		}
	default:
		fmt.Println("Payload variant is not 'packet'")
	}
	return nil
}

// debugPrintProtobuf takes a protobuf message and prints it in a pretty way for debugging
func debugPrintProtobuf(message proto.Message) {
	// Convert the Protobuf message to text format
	textData, err := prototext.MarshalOptions{
		Multiline: true, // Use multiline output for readability
	}.Marshal(message)

	if err != nil {
		fmt.Printf("Failed to marshal Protobuf message to text: %v", err)
		return
	}

	// Print the formatted Protobuf message
	fmt.Println("Protobuf message content (Text):")
	fmt.Println(string(textData))
}
