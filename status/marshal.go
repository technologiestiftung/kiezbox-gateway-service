package status

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

// Marshals KiexboxStatus message into a byte slice.
func MarshalKiezboxStatus(data *KiezboxStatus) ([]byte, error) {
	marshalledData, err := proto.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal SensorData: %w", err)
	}
	return marshalledData, nil
}

// Unmarshals the byte slice back into a KiezboxStatus message.
func UnmarshalKiezboxStatus(data []byte) (*KiezboxStatus, error) {
	var sensorData KiezboxStatus
	err := proto.Unmarshal(data, &sensorData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal SensorData: %w", err)
	}
	return &sensorData, nil
}
