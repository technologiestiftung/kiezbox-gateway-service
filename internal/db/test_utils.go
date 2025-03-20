package db

import (
	"time"

	influxdb "github.com/influxdata/influxdb-client-go/v2"
	influxdb_write "github.com/influxdata/influxdb-client-go/v2/api/write"
)

// Create a pre-defined InfluxDB Point
func CreateTestPoint() *influxdb_write.Point {
	return influxdb.NewPoint(
		// Measurement
		"sensor_data",
		// Tags
		map[string]string{
			"box_id":    "1",
			"district":  "1",
		},
		// Fields
		map[string]any{
			"router_powered":   true,
			"temperature_out":  float32(30),
			"temperature_in":   float32(28),
		},
		// Timestamp
		time.Unix(1672531200, 0),
	)
}

// Create an InfluxDB Point with a dynamic timestamp
func CreateTestPointDynamic() *influxdb_write.Point {
	return influxdb.NewPoint(
		// Measurement
		"sensor_data",
		// Tags
		map[string]string{
			"box_id":    "1",
			"district":  "1",
		},
		// Fields
		map[string]any{
			"router_powered":   true,
			"temperature_out":  float32(30),
			"temperature_in":   float32(28),
		},
		// Timestamp
		time.Unix(time.Now().Unix(), 0),
	)
}

// Create a valid InfluxDB query string
func createQuery(bucket string) string {
	return `
		from(bucket: "` + bucket + `")
			|> range(start: -1h)
			|> filter(fn: (r) => r["_measurement"] == "sensor_data")
			|> filter(fn: (r) => r["_field"] == "temperature_out" or r["_field"] == "temperature_in" or r["_field"] == "humidity_in")
			|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
			|> yield(name: "sensor_data")
	`
}
