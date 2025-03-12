package db

import (
	"context"
	"testing"
	"time"

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
	// Simulate a real delay to trigger context timeout
	if args.Error(0) == context.DeadlineExceeded {
		time.Sleep(Timeout + 1 * time.Second)
	}

	// Return the mocked error (or nil if successful)
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
			name:          "Database timeout",
			mockReturnErr: context.DeadlineExceeded,
			expectedErr:   "Database connection timed out",
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
				Timeout:  Timeout,
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
