package db

import (
	"context"
	"errors"
	"testing"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mocks
type MockQueryAPI struct {
	mock.Mock
}

func (m *MockQueryAPI) Query(ctx context.Context, query string) (*api.QueryTableResult, error) {
	args := m.Called(ctx, query)
	result, ok := args.Get(0).(*api.QueryTableResult)
	if !ok {
		return nil, args.Error(1)
	}
	return result, args.Error(1)
}

func (m *MockQueryAPI) QueryRaw(ctx context.Context, query string, dialect *domain.Dialect) (string, error) {
	args := m.Called(ctx, query, dialect)
	result, ok := args.Get(0).(string)
	if !ok {
		return "", args.Error(1)
	}
	return result, args.Error(1)
}

func (m *MockQueryAPI) QueryRawWithParams(ctx context.Context, query string, dialect *domain.Dialect, params interface{}) (string, error) {
	args := m.Called(ctx, query, dialect, params)
	result, ok := args.Get(0).(string)
	if !ok {
		return "", args.Error(1)
	}
	return result, args.Error(1)
}

func (m *MockQueryAPI) QueryWithParams(ctx context.Context, query string, params interface{}) (*api.QueryTableResult, error) {
	args := m.Called(ctx, query, params)
	result, ok := args.Get(0).(*api.QueryTableResult)
	if !ok {
		return nil, args.Error(1)
	}
	return result, args.Error(1)
}

func TestQueryData(t *testing.T) {
	// Define test cases
	testCases := []struct {
		name        string
		mockReturn  *api.QueryTableResult
		mockError   error
		expectedErr string
	}{
		{
			name:        "Success",
			mockReturn:  nil,
			mockError:   nil,
			expectedErr: "",
		},
		{
			name:        "Failure - query error",
			mockReturn:  nil,
			mockError:   errors.New("query failed"),
			expectedErr: "error executing query: query failed",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Arrange: Mock the QueryAPI behavior
			mockQueryAPI := new(MockQueryAPI)
			query := createQuery("test-bucket")
			mockQueryAPI.On("Query", mock.Anything, query).Return(testCase.mockReturn, testCase.mockError)

			// Initialize the mocked InfluxDB instance
			db := &InfluxDB{
				Client:   nil,          // Not needed for this test
				WriteAPI: nil,          // Not needed for this test
				QueryAPI: mockQueryAPI, // Use the mocked QueryAPI
				Org:      "test-org",
				Bucket:   "test-bucket",
			}

			// Act: Call QueryData
			_, err := db.QueryData(query)

			// Assert: Verify results based on the test case
			if testCase.expectedErr == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Equal(t, testCase.expectedErr, err.Error())
			}

			// Assert expectations on the mock
			mockQueryAPI.AssertExpectations(t)
		})
	}
}