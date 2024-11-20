package main

import (
	"fmt"
	"log"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	config "kiezbox/internal/config"
	db "kiezbox/internal/db"
	"kiezbox/internal/github.com/meshtastic/go/generated"
	marshal "kiezbox/internal/marshal" // TODO: is the alias redundant?
)

func main() {
	// --- Mock Kiezbox data, marshalling and unmarshalling ---
	
	// Step 1: Create a new KiezboxStatus message
	statusData := &generated.KiezboxMessage{
		Update: &generated.KiezboxMessage_Update{
			Meta: &generated.KiezboxMessage_Meta{
				BoxId:  1, // Example value
				DistId: 2, // Example value
			},
			UnixTime: time.Now().Unix(), // Current Unix timestamp
			Core: &generated.KiezboxMessage_Core{
				Mode: generated.KiezboxMessage_normal, // Required field
				Router: &generated.KiezboxMessage_Router{
					Powered: true, // Example value
				},
				Values: &generated.KiezboxMessage_CoreValues{}, // Required, but no optional fields set
			},
		},
	}

	// Marshal
	marshalledData, err := marshal.MarshalKiezboxMessage(statusData)
	if err != nil {
		log.Fatalf("Error marshalling data: %v", err)
	}

	fmt.Printf("Marshalled Data: %x\n", marshalledData)

	// Unmarshal
	unmarshalledData, err := marshal.UnmarshalKiezboxMessage(marshalledData)
	if err != nil {
		log.Fatalf("Error unmarshalling data: %v", err)
	}

	// --- Write data to the DB
	// Load InfluxDB configuration
	url, token, org, bucket := config.LoadConfig()

	// Initialize InfluxDB client
	db_client := db.CreateClient(url, token, org, bucket)
	defer db_client.Close()

	// Prepare the InfluxDB point
	point := influxdb2.NewPoint(
		// Measurement
		"sensor_data",
		// Tags
		map[string]string{
			"box_id":   fmt.Sprintf("%d", unmarshalledData.Update.Meta.BoxId),
			"district": fmt.Sprintf("%d", unmarshalledData.Update.Meta.DistId),
		},
		// Fields
		map[string]any{
			"router_powered":     unmarshalledData.Update.Core.Router.Powered,
			"temperature_out":    float32(unmarshalledData.Update.Core.Values.TempOut) / 1000, // Converting to float32 and °C
			"temperature_in":     float32(unmarshalledData.Update.Core.Values.TempIn) / 1000,
		},
		// Timestamp
		time.Unix(unmarshalledData.Update.UnixTime, 0),
	)

	// Write the point to InfluxDB
	if err := db_client.WriteData(point); err != nil {
		log.Fatalf("Error writing data: %v", err)
	}

	fmt.Println("Data written to InfluxDB successfully")

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
