package db

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	influxdb "github.com/influxdata/influxdb-client-go/v2"
	influxdb_write "github.com/influxdata/influxdb-client-go/v2/api/write"
	"google.golang.org/protobuf/reflect/protoreflect"

	"kiezbox/internal/github.com/meshtastic/go/generated"
	"kiezbox/internal/marshal"
)

const CacheDir = "kiezbox/internal/cached"

// WritePointToDatabase writes an InfluxDB point to the InfluxDB bucket
func (db *InfluxDB) WritePointToDatabase(point *influxdb_write.Point) error {
	// Set a timeout
	ctx, cancel := context.WithTimeout(context.Background(), db.Timeout)
	defer cancel()

	// Try writing in the database with context
	err := db.WriteAPI.WritePoint(ctx, point)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Println("Database connection timed out")
			return fmt.Errorf("database connection timed out: %w", ctx.Err())
		} else {
			log.Println("Data error: %w", err)
		}
	}
	return nil
}

// WritePointToFile writes a point as Protobuf message to a file in the given directory
func WritePointToFile(message *generated.KiezboxMessage, dir string) error {
	// Create a filename
	filename := fmt.Sprintf("%s.pb", uuid.New().String())

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory when caching: %w", err)
	}

	// Marshal the message
	marshalledMessage, err := marshal.MarshalKiezboxMessage(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message when caching: %w", err)
	}

	// Write to file
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, marshalledMessage, 0666); err != nil {
		return fmt.Errorf("failed to save cached message to file: %w", err)
	}

	return nil
}

// ReadPointFromFile reads a marshalled Protobuf message from a file and unmarshals it.
func ReadPointFromFile(filepath string) (*generated.KiezboxMessage, error) {
	// Read the file content
	marshalledMessage, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filepath, err)
	}

	// Unmarshal the Protobuf message
	message, err := marshal.UnmarshalKiezboxMessage(marshalledMessage)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal message from file %s: %w", filepath, err)
	}

	return message, nil
}

// RetryCachedPoints reads cached points from a directory and retries writing them to the database
func (db *InfluxDB) RetryCachedPoints(dir string) {
	// Read the directory
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("Failed to read directory %s: %v", dir, err)
	}

	// Iterate over the files and read the points
	for _, file := range files {
		filePath := filepath.Join(dir, file.Name())
		message, err := ReadPointFromFile(filePath) // Read and unmarshal Protobuf message
		if err != nil {
			log.Printf("Failed to read file %s: %v", filePath, err)
			continue
		}

		// Convert the Protobuf message to an InfluxDB point
		point, err := KiezboxMessageToPoint(message)
		if err != nil {
			log.Printf("Failed to convert message to point: %v", err)
			continue // Skip this file and move to the next
		}

		// Write the message to the database
		err = db.WritePointToDatabase(point)

		// Cache message if connection to database failed
		if err == nil || !errors.Is(err, context.DeadlineExceeded) {
			// Delete the file
			if err := os.Remove(filePath); err != nil {
				log.Printf("Failed to delete cached point %s: %v", filePath, err)
			} else {
				log.Printf("Successfully deleted cached point %s", filePath)
			}
		}
	}
}

func KiezboxMessageToPoint(message *generated.KiezboxMessage) (*influxdb_write.Point, error) {
	tags := make(map[string]string)
	fields := make(map[string]any)
	var measurement string

	// Iterate over the meta data and add them to the tags
	meta_reflect := message.Update.Meta.ProtoReflect()
	meta_reflect.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		// Get the meta tags
		tags[string(fd.Name())] = v.String()
		return true // Continue iteration
	})

	// Iterate over the values and add them to the fields
	if message.Update.Core != nil {
		measurement = "core_values"
		core_reflect := message.Update.Core.Values.ProtoReflect()
		core_reflect.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if intVal, ok := v.Interface().(int32); ok {
				fields[string(fd.Name())] = float64(intVal) / 1000.0
			} else {
				fmt.Printf("Unexpected type for field %s: %T\n", fd.Name(), v.Interface())
			}
			return true // Continue iteration
		})
	} else if message.Update.Sensor != nil {
		measurement = "sensor_values"
		sensor_reflect := message.Update.Sensor.Values.ProtoReflect()
		sensor_reflect.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if intVal, ok := v.Interface().(int32); ok {
				fields[string(fd.Name())] = float64(intVal) / 1000.0
			} else {
				fmt.Printf("Unexpected type for field %s: %T\n", fd.Name(), v.Interface())
			}
			return true // Continue iteration
		})
	}

	// Add an additional field with the gateway systems arrival time
	// Currently used for debugging and sanity checking
	fields["time_arrival"] = time.Now().Format(time.RFC3339)

	// Prepare the InfluxDB point
	point := influxdb.NewPoint(
		// Measurement
		measurement,
		// Tags
		tags,
		// Fields
		fields,
		// Timestamp
		time.Unix(message.Update.UnixTime, 0),
	)

	return point, nil
}
