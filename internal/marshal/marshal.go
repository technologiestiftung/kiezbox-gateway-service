package marshal

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	"kiezbox/internal/github.com/meshtastic/go/generated"
)

// Marshals KiexboxStatus message into a byte slice.
func MarshalKiezboxMessage(data *generated.KiezboxMessage) ([]byte, error) {
	if data == nil {
		return nil, fmt.Errorf("empty or invalid KiezboxMessage")
	}
	marshalledData, err := proto.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SensorData: %w", err)
	}
	return marshalledData, nil
}

// Unmarshals the byte slice back into a KiezboxStatus message.
func UnmarshalKiezboxMessage(data []byte) (*generated.KiezboxMessage, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data to unmarshal")
	}
	var sensorData generated.KiezboxMessage
	err := proto.Unmarshal(data, &sensorData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal SensorData: %w", err)
	}
	return &sensorData, nil
}
