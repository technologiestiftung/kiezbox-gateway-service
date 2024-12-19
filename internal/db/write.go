package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/reflect/protoreflect"

	"kiezbox/internal/github.com/meshtastic/go/generated"

	influxdb "github.com/influxdata/influxdb-client-go/v2"
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

// KiezboxMessage2Point converts a KiezboxMessage to an InfluxDB point
func KiezboxMessage2Point(message *generated.KiezboxMessage) *influxdb_write.Point {
	// TODO: Change prints to logs
	fmt.Println("Handling Protobuf message")

	// Prepare point structure
	var measurement string
	tags := make(map[string]string)
	fields := make(map[string]any)
	var timestamp time.Time

	// Iterate over the meta data and add them to the tags
	// Validate that mandatory meta data is present and of the right type
	if message.Update.Meta != nil {
		if message.Update.Meta.BoxId != nil {
			if _, ok := interface{}(message.Update.Meta.BoxId).(*uint32); !ok {
				fmt.Errorf("Meta field BoxId is of the wrong type: expected *uint32, got %T\n", message.Update.Meta.BoxId)
			}
		} else {
			errors.New("Meta field 'BoxId' is missing")
		}
		if message.Update.Meta.DistId != nil {
			if _, ok := interface{}(message.Update.Meta.DistId).(*uint32); !ok {
				fmt.Errorf("Meta field DistId is of the wrong type: expected *uint32, got %T\n", message.Update.Meta.DistId)
			}
		} else {	
			errors.New("Meta field 'DistId' is missing")
		}
		// Add the meta data to the tags
		meta_reflect := message.Update.Meta.ProtoReflect()
		meta_reflect.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			// Get the meta tags
			tags[string(fd.Name())] = v.String()
			return true // Continue iteration
		})
	} else {
		errors.New("Field 'Meta' is missing")
	}

	// Iterate over the Core and Sensor values and add them to the fields
	if message.Update.Core != nil {
		measurement = "core_values"
		core_reflect := message.Update.Core.Values.ProtoReflect()
		core_reflect.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if intVal, ok := v.Interface().(int32); ok {
				fields[string(fd.Name())] = float64(intVal) / 1000.0
			} else {
				fmt.Errorf("Unexpected type for field %s: %T\n", fd.Name(), v.Interface())
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
				fmt.Errorf("Unexpected type for field %s: %T\n", fd.Name(), v.Interface())
			}
			return true // Continue iteration
		})
	}

	// Add an additional field with the gateway systems arrival time
	// Currently used for debugging and sanity checking
	fields["time_arrival"] = time.Now().Format(time.RFC3339)

	// Set and validate the timestamp
	if message.Update.UnixTime != 0 {
		if _, ok := interface{}(message.Update.UnixTime).(int64); !ok {
			fmt.Errorf("Meta field BoxId is of the wrong type: expected int64, got %T\n", message.Update.UnixTime)
		} else {
			timestamp = time.Unix(message.Update.UnixTime, 0)
		} 
	} else {
		errors.New("Field 'KiezboxMessage.Update.UnixTime' is missing")
	}

	// Prepare the InfluxDB point
	point := influxdb.NewPoint(
		// Measurement
		measurement,
		// Tags
		tags,
		// Fields
		fields,
		// Timestamp
		timestamp,
	)
	fmt.Printf("Addint point: %+v\n", point)

	return point
}