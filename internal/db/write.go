package db

import (
	"context"
	"fmt"

	influxdb_write "github.com/influxdata/influxdb-client-go/v2/api/write"
)

// WriteData writes a point to the InfluxDB bucket
func (db *InfluxDB) WriteData(point *influxdb_write.Point) error {
	// Set a timeout
	ctx, cancel := context.WithTimeout(context.Background(), db.Timeout)
	defer cancel()

	// Try writing in the database with context
	err := db.WriteAPI.WritePoint(ctx, point)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("Database connection timed out")
		} else {
			return fmt.Errorf("Data error: %w", err)
		}
	}
	return nil
}
