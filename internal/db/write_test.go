package db

import (
	"context"
	"encoding/gob"
	"os"
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
		expectedLog   string
	}{
		{
			name:          "Success",
			mockReturnErr: nil,
			expectedErr:   "",
			expectedLog:   "",
		},
		{
			name:          "Database timeout",
			mockReturnErr: context.DeadlineExceeded,
			expectedErr:   "",
			expectedLog:   "Database connection timed out",
		},
		// TODO: Write a test for non valid data
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

func TestWriteDataWithTimeout(t *testing.T) {
	// Setup a mock WriteAPI to simulate timeout
	mockWriteAPI := new(MockWriteAPI)
	mockWriteAPI.On("WritePoint", mock.Anything, mock.Anything).Return(context.DeadlineExceeded)

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

	// Call WriteData to simulate timeout and file write
	err := db.WriteData(point)

	// Ensure no error occurs
	assert.NoError(t, err)

	// Check if the .gob file exists
	_, err = os.Stat("cached_datapoints.gob")
	assert.NoError(t, err, "Expected .gob file to exist after timeout")

	// Read and decode the last written point from the .gob file
	file, err := os.Open("cached_datapoints.gob")
	assert.NoError(t, err, "Failed to open .gob file")
	defer file.Close()

	decoder := gob.NewDecoder(file)

	// Decode into SerializedPoint first
	var decodedSerializedPoint SerializedPoint
	err = decoder.Decode(&decodedSerializedPoint)
	assert.NoError(t, err, "Failed to decode .gob file")

	// Convert SerializedPoint back to influxdb_write.Point
	decodedPoint := UnserializePoint(&decodedSerializedPoint)

	// Compare the original point and the decoded point directly
	assert.Equal(t, point.Name(), decodedPoint.Name(), "Measurement does not match")

	// Convert the point tags to a map for comparison using TagList()
	originalTags := make(map[string]string)
	for _, tag := range point.TagList() {
		originalTags[tag.Key] = tag.Value
	}
	// Convert the tags of decodedPoint to map for comparison
	decodedTags := make(map[string]string)
	for _, tag := range decodedPoint.TagList() {
		decodedTags[tag.Key] = tag.Value
	}

	assert.Equal(t, originalTags, decodedTags, "Tags do not match")

	// Convert the point fields to a map for comparison
	originalFields := make(map[string]interface{})
	for _, field := range point.FieldList() {
		originalFields[field.Key] = field.Value
	}

	// Convert the fields of decodedPoint to a map for comparison
	decodedFields := make(map[string]interface{})
	for _, field := range decodedPoint.FieldList() {
		decodedFields[field.Key] = field.Value
	}

	assert.Equal(t, originalFields, decodedFields, "Fields do not match")

	// Compare time
	assert.Equal(t, point.Time(), decodedPoint.Time(), "Time does not match")
}
