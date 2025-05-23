package db

import (
	"context"
	"errors"
	"testing"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"kiezbox/testutils"
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
			// Initialize and set behavior of QueryAPI mock
			mockQueryAPI := new(MockQueryAPI)
			query := testutils.CreateQuery("test-bucket")
			mockQueryAPI.On("Query", mock.Anything, query).Return(testCase.mockReturn, testCase.mockError)

			//  Initialize InfluxDB instance mock
			db := &InfluxDB{
				Client:   nil,
				WriteAPI: nil,
				QueryAPI: mockQueryAPI,
				Org:      "test-org",
				Bucket:   "test-bucket",
			}

			// Call QueryData
			_, err := db.QueryData(query)

			// Verify results based on the test case
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
