package db

import (
	"context"
	"fmt"

	influxdb_query "github.com/influxdata/influxdb-client-go/v2/api"
)

// QueryData retrieves data from the InfluxDB bucket
func (db *InfluxDB) QueryData(query string) (*influxdb_query.QueryTableResult, error) {
	result, err := db.QueryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	return result, nil
}
