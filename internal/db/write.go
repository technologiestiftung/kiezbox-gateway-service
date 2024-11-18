package db

import (
	"context"
	"fmt"

	influxdb_write "github.com/influxdata/influxdb-client-go/v2/api/write"
)

// WriteData writes a point to the InfluxDB bucket
func (db *InfluxDB) WriteData(point *influxdb_write.Point) error {
	err := db.WriteAPI.WritePoint(context.Background(), point)
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	return nil
}
