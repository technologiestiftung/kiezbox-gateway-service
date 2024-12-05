package marshal

import (
	"testing"

	"kiezbox/internal/github.com/meshtastic/go/generated"

	"github.com/stretchr/testify/assert"
)

func TestMarshalKiezboxMessageValid(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name          string
		kiezboxMessage *generated.KiezboxMessage
		expectError   bool
	}{
		{
			name:          "Successful Marshalling",
			kiezboxMessage: CreateKiezboxMessage(),
			expectError:   false,
		},
		{
			name:          "Empty Message",
			kiezboxMessage: &generated.KiezboxMessage{}, // Invalid message to trigger an error
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal the KiezboxMessage
			marshalledData, err := MarshalKiezboxMessage(tc.kiezboxMessage)

			// Check if the error condition matches the expectation
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, marshalledData)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, marshalledData)
			}
		})
	}
}

func TestUnmarshalKiezboxMessageValid(t *testing.T) {
	// Table-driven tests
	testCases := []struct {
		name          string
		kiezboxMessage *generated.KiezboxMessage
		expectError   bool
	}{
		{
			name:          "Successful Unmarshalling",
			kiezboxMessage: CreateKiezboxMessage(),
			expectError:   false,
		},
		{
			name:          "Invalid Data (corrupted)",
			kiezboxMessage: nil, // Passing invalid data to trigger an error
			expectError:   true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Marshal the KiezboxMessage to create byte slice
			var marshalledData []byte
			if tc.kiezboxMessage != nil {
				var err error
				marshalledData, err = MarshalKiezboxMessage(tc.kiezboxMessage)
				assert.NoError(t, err)
			}

			// Unmarshal the marshalledData
			unmarshalledData, err := UnmarshalKiezboxMessage(marshalledData)

			// Check if the error condition matches the expectation
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, unmarshalledData)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, unmarshalledData)
				// Compare original data and unmarshalled data
				assert.Equal(t, tc.kiezboxMessage.Update.Meta.BoxId, unmarshalledData.Update.Meta.BoxId)
				assert.Equal(t, tc.kiezboxMessage.Update.Meta.DistId, unmarshalledData.Update.Meta.DistId)
			}
		})
	}
}

func TestUnmarshalKiezboxMessageInvalid(t *testing.T) {
	// Table-driven tests for invalid unmarshalling
	testCases := []struct {
		name        string
		data        []byte
		expectError bool
	}{
		{
			name:        "Corrupted Data",
			data:        []byte("invalid_data"), // Invalid byte slice
			expectError: true,
		},
		{
			name:        "Empty Data",
			data:        []byte{}, // Empty byte slice
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Try unmarshalling the invalid data
			unmarshalledData, err := UnmarshalKiezboxMessage(tc.data)

			// Check if error occurred as expected
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, unmarshalledData)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, unmarshalledData)
			}
		})
	}
}
