package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	influxdb_write "github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"google.golang.org/protobuf/proto"

	"kiezbox/internal/github.com/meshtastic/go/generated"
	"kiezbox/testutils"
)

// Set the test timeout to a very short duration
// to ensure that the context timeout is triggered quickly
const testTimeout = 1 * time.Nanosecond

// WriteAPI Mocks
type MockWriteAPI struct {
	mock.Mock
}

func (m *MockWriteAPI) WritePoint(ctx context.Context, point ...*influxdb_write.Point) error {
	args := m.Called(ctx, point)
	// Simulate a real delay to trigger context timeout
	if args.Error(0) == context.DeadlineExceeded {
		time.Sleep(testTimeout + 1*time.Second)
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

// MarshalKiexboxMessage mocks
type MockMarshal struct {
	mock.Mock
}

func (m *MockMarshal) MarshalKiezboxMessage(message *generated.KiezboxMessage) ([]byte, error) {
	args := m.Called(message)

	// Simulate a marshaling error if mockReturnErr is set
	if args.Error(0) != nil {
		// If the test case provides an error, return it
		return nil, args.Error(0)
	}

	// Otherwise, return the marshaled message as expected
	return proto.Marshal(message)
}

// InfluxDB Mocks
type MockInfluxDB struct {
	mock.Mock
}

func (m *MockInfluxDB) WritePointToDatabase(point *influxdb_write.Point) error {
	args := m.Called(point)
	// Simulate a real delay to trigger context timeout
	if args.Error(0) == context.DeadlineExceeded {
		time.Sleep(testTimeout + 1*time.Second)
	}
	// Return the mocked error (or nil if successful)
	return args.Error(0)
}

func TestWritePointToDatabase(t *testing.T) {
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
			expectedErr:   "database connection timed out: context deadline exceeded",
			expectedLog:   "database connection timed out: context deadline exceeded",
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
				Timeout:  testTimeout,
			}

			// Prepare the InfluxDB point
			point := testutils.CreateTestPoint()

			// Call WritePointToDatabase
			err := db.WritePointToDatabase(point)

			// Verify results based on the test cases
			if testCase.expectedErr == "" {
				assert.NoError(t, err) // No error expected
			} else {
				assert.Error(t, err)                               // Error expected
				assert.Equal(t, testCase.expectedErr, err.Error()) // Verify error message
			}

			// Assert expectations on the mock
			mockWriteAPI.AssertExpectations(t)
		})
	}
}

func TestWritePointToFile(t *testing.T) {
	// Create a kiezbox message to use throughout the test
	testMessage := testutils.CreateKiezboxMessage(time.Now().Unix())
	filename := fmt.Sprintf("%d.pb", testMessage.Update.UnixTime)

	tests := []struct {
		name          string
		message       *generated.KiezboxMessage
		dir           string
		mockReturnErr error
		expectedErr   error
	}{
		{
			name:          "Successful write",
			message:       testMessage,
			dir:           "", // Will be replaced with t.TempDir()
			mockReturnErr: nil,
			expectedErr:   nil,
		},
		// TODO: Add test cases for the different failure scenarios
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// Use a real temporary directory instead of faking os.MkdirAll
			dir := t.TempDir()

			// Initialize the MockMarshal object
			mockMarshal := new(MockMarshal)

			// Set the behavior of the mock for marshaling
			mockMarshal.On("MarshalKiezboxMessage", mock.Anything).Return(nil, testCase.mockReturnErr)

			// Call the function
			err := WritePointToFile(testCase.message, dir)

			// Check the results
			if testCase.expectedErr != nil {
				assert.EqualError(t, err, testCase.expectedErr.Error())
			} else {
				assert.NoError(t, err)

				// Construct the expected file path
				expectedFilePath := filepath.Join(dir, filename)

				// Check if the file was created with the expected name
				assert.FileExists(t, expectedFilePath, "File was not created with the expected name")
			}
		})
	}
}

// Helper function to copy the fixture files into the temp directory
func copyFixtureFiles(t *testing.T, sourceDir, destDir string) {
	// Read the source directory to get the list of fixture files
	files, err := os.ReadDir(sourceDir)
	if err != nil {
		t.Fatalf("Failed to read source directory %s: %v", sourceDir, err)
	}

	// Iterate over files and copy them
	for _, file := range files {
		// Get the full source and destination file paths
		sourcePath := filepath.Join(sourceDir, file.Name())
		destPath := filepath.Join(destDir, file.Name())

		// Read the content of the source file
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatalf("Failed to read fixture file %s: %v", sourcePath, err)
		}

		// Write the content into the temporary directory
		err = os.WriteFile(destPath, content, 0666)
		if err != nil {
			t.Fatalf("Failed to write file to temporary directory %s: %v", destPath, err)
		}
	}
}

func TestRetryCachedPoints(t *testing.T) {
	// Define the directory containing the fixtures
	originalDir := "fixtures/cached"

	// Temporary directory to copy the fixture files
	tempDir := t.TempDir()

	// Define the test cases
	tests := []struct {
		name                string
		mockReturnErr       error
		expectedFileDeleted bool
	}{
		{
			name:                "Success: Files should be deleted after successful write",
			mockReturnErr:       nil,
			expectedFileDeleted: true,
		},
		{
			name:                "Timeout: Files should not be deleted on DeadlineExceeded error",
			mockReturnErr:       context.DeadlineExceeded,
			expectedFileDeleted: false,
		},
	}

	// Iterate over test cases
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// Always copy fixture files into the temporary directory before each test
			copyFixtureFiles(t, originalDir, tempDir)

			// Mock the behavior of mocks
			mockWriteAPI := new(MockWriteAPI)
			mockWriteAPI.On("WritePoint", mock.Anything, mock.Anything).Return(testCase.mockReturnErr)

			// Setup the InfluxDB instance mock
			db := &InfluxDB{
				Client:   nil,
				WriteAPI: mockWriteAPI,
				QueryAPI: nil,
				Org:      "test-org",
				Bucket:   "test-bucket",
				Timeout:  testTimeout,
			}

			// Get the filenames from the directory
			files, err := os.ReadDir(tempDir)
			if err != nil {
				t.Fatalf("Failed to read tempDir %s: %v", tempDir, err)
			}

			// Ensure that the files exist before the test
			for _, file := range files {
				assert.FileExists(t, filepath.Join(tempDir, file.Name()))
			}

			// Run RetryCachedPoints
			db.RetryCachedPoints(tempDir)

			// Verify the files' existence based on the result of WritePointToDatabase
			for _, file := range files {
				filePath := filepath.Join(tempDir, file.Name())

				if testCase.expectedFileDeleted {
					// If the write was successful, the files should be deleted
					assert.NoFileExists(t, filePath)
				} else {
					// If there was a timeout, the files should still exist
					assert.FileExists(t, filePath)
				}
			}
		})
	}
}
