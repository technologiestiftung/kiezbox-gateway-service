package main

import (
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"google.golang.org/protobuf/reflect/protoreflect"
	"kiezbox/internal/config"
	"kiezbox/internal/db"
	"kiezbox/internal/meshtastic"
	"log"
	"time"
)

func main() {
	// Buffered channel for handling Protobuf messages
	var mts meshtastic.MTSerial
	mts.Init("/dev/ttyUSB0", 115200)

	// Launch a goroutine for serial reading.
	go mts.Writer()
	go mts.Heartbeat(30 * time.Second)
	go mts.Reader()

	// Load InfluxDB configuration
	url, token, org, bucket := config.LoadConfig()

	// Initialize InfluxDB client
	db_client := db.CreateClient(url, token, org, bucket)
	defer db_client.Close()

	// Process Protobuf messages in the main goroutine.
	//TODO: move this into it's own gorouting
	for FromRadio := range mts.FromChan {
		message := meshtastic.ExtractKBMessage(FromRadio)
		if message == nil {
			continue
		}
		fmt.Println("Handling Protobuf message")
		tags := make(map[string]string)
		fields := make(map[string]any)
		meta_reflect := message.Update.Meta.ProtoReflect()
		meta_reflect.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			// Get the meta tags
			tags[string(fd.Name())] = v.String()
			return true // Continue iteration
		})
		core_reflect := message.Update.Core.Values.ProtoReflect()
		core_reflect.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			if intVal, ok := v.Interface().(int32); ok {
				fields[string(fd.Name())] = float64(intVal) / 1000.0
			} else {
				fmt.Printf("Unexpected type for field %s: %T\n", fd.Name(), v.Interface())
			}
			return true // Continue iteration
		})
		// Add an additional field with the gateway systems arrival time
		// Currently used for debugging and sanity checking
		fields["time_arrival"] = time.Now().Format(time.RFC3339)
		// Prepare the InfluxDB point
		point := influxdb2.NewPoint(
			// Measurement
			"core_values",
			// Tags
			tags,
			// Fields
			fields,
			// Timestamp
			time.Unix(message.Update.UnixTime, 0),
		)
		fmt.Printf("Addint point: %+v\n", point)

		// Write the point to InfluxDB
		err := db_client.WriteData(point)
		fmt.Println("Writing data... Err?", err)

		fmt.Println("Data written to InfluxDB successfully")
	}

	// --- Retrieve Data from InfluxDB ---
	// Define a Flux query to retrieve sensor data
	query := fmt.Sprintf(`
		from(bucket: "%s")
			|> range(start: -1h)  // Retrieve data from the last 1 hour
			|> filter(fn: (r) => r["_measurement"] == "sensor_data")
			|> filter(fn: (r) => r["_field"] == "temperature_out" or r["_field"] == "temperature_in" or r["_field"] == "humidity_in")
			|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
			|> yield(name: "sensor_data")
	`, bucket)

	// Execute the query
	result, err := db_client.QueryData(query)
	if err != nil {
		log.Fatalf("Error querying data: %v", err)
	}

	// Iterate over the query result and print the data
	for result.Next() {
		// Access the returned record
		fmt.Printf("Time: %s\n", result.Record().Time())
		fmt.Printf("Box ID: %s, District ID: %s\n", result.Record().ValueByKey("box_id"), result.Record().ValueByKey("district"))
		fmt.Printf("Temperature Outside: %.2f°C\n", result.Record().ValueByKey("temperature_out"))
		fmt.Printf("Temperature Inside: %.2f°C\n", result.Record().ValueByKey("temperature_in"))
		fmt.Printf("Humidity Inside: %.2f%%\n", result.Record().ValueByKey("humidity_in"))
	}

	// Check for errors in the query results
	if result.Err() != nil {
		log.Fatalf("Query failed: %v", result.Err())
	}

	fmt.Println("Data retrieved from InfluxDB successfully")
}
