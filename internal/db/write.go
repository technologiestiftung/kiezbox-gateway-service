package db

import (
	"context"
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"time"

	influxdb_write "github.com/influxdata/influxdb-client-go/v2/api/write"
)

type SerializedPoint struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]interface{}
	Time        time.Time
}

// WriteData writes a point to the InfluxDB bucket
func (db *InfluxDB) WriteData(point *influxdb_write.Point) error {
	// Set a timeout
	ctx, cancel := context.WithTimeout(context.Background(), db.Timeout)
	defer cancel()

	// Try writing in the database with context
	err := db.WriteAPI.WritePoint(ctx, point)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("Database connection timed out")

			// Write the point to a .gob file
			if err := WritePointToGobFile(point); err != nil {
				log.Printf("Error writing point to gob file: %v", err)
				return fmt.Errorf("Error writing point to gob file: %w", err)
			}

		} else {
			log.Println("Data error: %w", err)
		}
	}
	return nil
}

// WritePointToGobFile serializes the point and writes it to a gob file
func WritePointToGobFile(point *influxdb_write.Point) error {
	// Open or create the .gob file
	file, err := os.OpenFile("cached_datapoints.gob", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Error opening .gob file: %w", err)
	}
	defer file.Close()

	// Convert to a serialized struct
	serializedPoint := SerializePoint(point)

	// Create a new encoder and write the point to the file
	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(serializedPoint); err != nil {
		return fmt.Errorf("Error encoding point to .gob file: %w", err)
	}

	return nil
}

// Convert influxdb_write.Point to SerializedPoint
func SerializePoint(point *influxdb_write.Point) SerializedPoint {
	// Convert tag list from []*protocol.Tag to map[string]string
	tags := make(map[string]string)
	for _, tag := range point.TagList() {
		tags[tag.Key] = tag.Value
	}

	// Convert field list from []*protocol.Field to map[string]interface{}
	fields := make(map[string]interface{})
	for _, field := range point.FieldList() {
		fields[field.Key] = field.Value
		}

	return SerializedPoint{
		Measurement: point.Name(),
		Tags:        tags,
		Fields:      fields,
		Time:        point.Time(),
	}
}

// Convert SerializedPoint to influxdb_write.Point
func UnserializePoint(sp *SerializedPoint) *influxdb_write.Point {
	return influxdb_write.NewPoint(sp.Measurement, sp.Tags, sp.Fields, sp.Time)
}
