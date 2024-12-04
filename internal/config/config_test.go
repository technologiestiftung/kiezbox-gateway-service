package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Define test cases
	tests := []struct {
		name          string
		envOverrides  map[string]string
		expectedError bool
	}{
		{
			name: "Valid configuration",
			envOverrides: map[string]string{
				"INFLUXDB_URL":     "http://localhost:8086",
				"INFLUXDB_TOKEN":   "valid-token",
				"INFLUXDB_ORG":     "org",
				"INFLUXDB_BUCKET":  "bucket",
			},
			expectedError: false,
		},
		{
			name: "Missing INFLUXDB_URL",
			envOverrides: map[string]string{
				"INFLUXDB_TOKEN":  "valid-token",
				"INFLUXDB_ORG":    "org",
				"INFLUXDB_BUCKET": "bucket",
			},
			expectedError: true,
		},
		{
			name: "Missing INFLUXDB_TOKEN",
			envOverrides: map[string]string{
				"INFLUXDB_URL":    "http://localhost:8086",
				"INFLUXDB_ORG":    "org",
				"INFLUXDB_BUCKET": "bucket",
			},
			expectedError: true,
		},
		{
			name: "Missing INFLUXDB_ORG",
			envOverrides: map[string]string{
				"INFLUXDB_URL":    "http://localhost:8086",
				"INFLUXDB_TOKEN":  "valid-token",
				"INFLUXDB_BUCKET": "bucket",
			},
			expectedError: true,
		},
		{
			name: "Missing INFLUXDB_BUCKET",
			envOverrides: map[string]string{
				"INFLUXDB_URL":   "http://localhost:8086",
				"INFLUXDB_TOKEN": "valid-token",
				"INFLUXDB_ORG":   "org",
			},
			expectedError: true,
		},
		{
			name:          "Missing all environment variables",
			envOverrides:  map[string]string{},
			expectedError: true,
		},
	}

	// Loop through each test case
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the environment variables for the test
			for key, value := range tt.envOverrides {
				err := os.Setenv(key, value)
				require.NoError(t, err)
			}

			// Load the config
			if tt.expectedError {
				// Expect the function to panic if an error occurs
				require.Panics(t, func() {
					LoadConfig()
				})
			} else {
				// Expect no error, so capture the returned values
				url, token, org, bucket := LoadConfig()

				// Validate the returned values
				assert.Equal(t, "http://localhost:8086", url)
				assert.Equal(t, "valid-token", token)
				assert.Equal(t, "org", org)
				assert.Equal(t, "bucket", bucket)
			}

			// Clean up environment variables for the next test
			for key := range tt.envOverrides {
				err := os.Unsetenv(key)
				require.NoError(t, err)
			}
		})
	}
}
