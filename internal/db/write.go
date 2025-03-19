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

const CachedDataFile = "cached_datapoints.gob"

type SerializedPoint struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]interface{}
	Time        time.Time
}

// WritePointToDatabase writes a point to the InfluxDB bucket
func (db *InfluxDB) WritePointToDatabase(point *influxdb_write.Point) error {
	// Set a timeout
	ctx, cancel := context.WithTimeout(context.Background(), db.Timeout)
	defer cancel()

	// Try writing in the database with context
	err := db.WriteAPI.WritePoint(ctx, point)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("Database connection timed out")

			// Write the point to a .gob file
			if err := WritePointToGobFile(CachedDataFile, point); err != nil {
				log.Printf("Error writing point to gob file: %v", err)
				return fmt.Errorf("Error writing point to gob file: %w", err)
			}

		} else {
			log.Println("Data error: %w", err)
		}
	}
	return nil
}

// WriteCachedPointsToDatabase writes a batch of cached points to the InfluxDB bucket and returns points that couldn't be written.
func (db *InfluxDB) WriteCachedPointsToDatabase(points []*influxdb_write.Point) []*influxdb_write.Point {
    // Set a timeout
    ctx, cancel := context.WithTimeout(context.Background(), db.Timeout)
    defer cancel()

    // Slice to collect points that couldn't be written
    var remainingPoints []*influxdb_write.Point

    // Iterate over the list of points and write each one individually
    for _, point := range points {
        // Try writing each point to the database with context
        err := db.WriteAPI.WritePoint(ctx, point)
        if err != nil {
            if ctx.Err() == context.DeadlineExceeded {
                log.Println("Database connection timed out")
            } else {
                log.Printf("Data error: %v", err)
            }
            // Add the point to the remaining points slice
            remainingPoints = append(remainingPoints, point)
        }
    }

    // Return the points that couldn't be written
    return remainingPoints
}

// WritePointToGobFile serializes the point and writes it to a gob file
func WritePointToGobFile(filename string, point *influxdb_write.Point) error {
	// Open or create the .gob file
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Error opening .gob file: %w", err)
	}
	defer file.Close()

	// Create a new encoder and Write the point to the file
	encoder := gob.NewEncoder(file)

	// Convert to a serialized struct
	serializedPoint := SerializePoint(point)

	// Write the point to the file
	if err := encoder.Encode(serializedPoint); err != nil {
		return fmt.Errorf("Error encoding point to .gob file: %w", err)
	}

	return nil
}

// WritePointsToGobFile serializes a batch of points and writes them to a gob file
// If overwrite is true, it will overwrite the file. If false, it will append to the file.
func WritePointsToGobFile(filename string, points []*influxdb_write.Point, overwrite bool) error {
	var file *os.File
	var err error

	// Determine the file open mode based on the overwrite flag
	if overwrite {
		// Open the .gob file with the O_WRONLY and O_TRUNC flags to overwrite the file
		file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	} else {
		// Open the .gob file with the O_APPEND flag to append to the file
		file, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	}

	if err != nil {
		return fmt.Errorf("Error opening .gob file: %w", err)
	}
	defer file.Close()

	// Create a new encoder
	encoder := gob.NewEncoder(file)

	// Iterate through all points, serialize each and write to the file
	for _, point := range points {
		// Convert to a serialized struct
		serializedPoint := SerializePoint(point)

		// Write the serialized point to the file
		if err := encoder.Encode(serializedPoint); err != nil {
			return fmt.Errorf("Error encoding point to .gob file: %w", err)
		}
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

// ReadPointsFromGobFile reads all cached points from the gob file and unserializes them into influxdb_write.Point
func ReadPointsFromGobFile(filename string) ([]*influxdb_write.Point, error) {
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No file means no cached data
		}
		return nil, fmt.Errorf("error opening .gob file: %w", err)
	}
	defer file.Close()

	var points []*influxdb_write.Point
	decoder := gob.NewDecoder(file)
	for {
		var serializedPoint SerializedPoint
		if err := decoder.Decode(&serializedPoint); err != nil {
			if err.Error() == "EOF" {
				break // End of file reached
			}
			return nil, fmt.Errorf("error decoding .gob file: %w", err)
		}

		// Unserialize the point and append it to the points slice
		point := UnserializePoint(&serializedPoint)
		points = append(points, point)
	}
	return points, nil
}


func (db *InfluxDB) RetryCachedPoints(filename string) {
    // Check if the file exists
    if _, err := os.Stat(filename); os.IsNotExist(err) {
        log.Printf("Cached file does not exist: %v", filename)
        return
    }

    // Read cached points from the .gob file
    cachedPoints, err := ReadPointsFromGobFile(filename)
    if err != nil {
        log.Printf("Error reading cached points: %v", err)
        return
    }

    if len(cachedPoints) == 0 {
        return // No cached points to process
    }

    // Try sending cached points to the database
    remainingPoints := db.WriteCachedPointsToDatabase(cachedPoints)

    // If all points were written, delete the .gob file
    if len(remainingPoints) == 0 {
        if err := os.Remove(filename); err != nil {
            log.Printf("Error deleting cached file: %v", err)
        }
        return
    }

    // Otherwise, update the .gob file with the remaining points
    if err := WritePointsToGobFile(filename, remainingPoints, true); err != nil {
        log.Printf("Error updating cached points: %v", err)
    }
}
