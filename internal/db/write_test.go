package db

import (
	"context"
	"errors"
	"testing"

	influxdb_write "github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks
type MockWriteAPI struct {
	mock.Mock
}

func (m *MockWriteAPI) WritePoint(ctx context.Context, point ...*influxdb_write.Point) error {
	args := m.Called(ctx, point)
	return args.Error(0)
}

func (m *MockWriteAPI) WriteRecord(ctx context.Context, line ...string) error {
	args := m.Called(ctx, line)
	return args.Error(0)
}

func (m *MockWriteAPI) EnableBatching() {
	m.Called()
}

func (m *MockWriteAPI) Flush(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}


func TestWriteData(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name          string
		mockReturnErr error
		expectedErr   string
	}{
		{
			name:          "Success",
			mockReturnErr: nil,
			expectedErr:   "",
		},
		{
			name:          "Failure - network error",
			mockReturnErr: errors.New("network error"),
			expectedErr:   "failed to write data: network error",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Initialize and set behavior of WriteAPI mock
			mockWriteAPI := new(MockWriteAPI)
			mockWriteAPI.On("WritePoint", mock.Anything, mock.Anything).Return(testCase.mockReturnErr)

            // Initialize InfluxDB instance mock
            db := &InfluxDB{
                Client:   nil,
                WriteAPI: mockWriteAPI,
                QueryAPI: nil,
                Org:      "test-org",
                Bucket:   "test-bucket",
            }

			// Prepare the InfluxDB point
			point := CreateTestPoint()

			// Call WriteData
			err := db.WriteData(point)

			// Verify results based on the test cases
			if testCase.expectedErr == "" {
				assert.NoError(t, err) // No error expected
			} else {
				assert.Error(t, err) // Error expected
				assert.Equal(t, testCase.expectedErr, err.Error()) // Verify error message
			}

			// Assert expectations on the mock
			mockWriteAPI.AssertExpectations(t)
		})
	}
}
