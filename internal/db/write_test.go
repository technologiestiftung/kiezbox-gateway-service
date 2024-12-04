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
	// TODO: Check if there are more types of errors to be handled
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
			// Arrange: Mock the WriteAPI behavior
			mockWriteAPI := new(MockWriteAPI)
			mockWriteAPI.On("WritePoint", mock.Anything, mock.Anything).Return(testCase.mockReturnErr)

			// TODO: Maybe move this somewhere else too, as QueryAPITest will also use it
            // Initialize the mocked InfluxDB instance
            db := &InfluxDB{
                Client:   nil,               // Not needed for this test
                WriteAPI: mockWriteAPI,      // Use the mocked WriteAPI
                QueryAPI: nil,               // Not needed for this test
                Org:      "test-org",        // Example organization
                Bucket:   "test-bucket",     // Example bucket
            }

			// Prepare the InfluxDB point
			point := CreateTestPoint()

			// Act: Call WriteData
			err := db.WriteData(point)

			// Assert: Verify results based on the test case
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
