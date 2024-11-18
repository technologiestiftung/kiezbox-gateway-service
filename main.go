package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/joho/godotenv"

	"kiezbox/status"
)

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Fetch InfluxDB configuration from environment variables
	url := os.Getenv("INFLUXDB_URL")
	token := os.Getenv("INFLUXDB_TOKEN")
	org := os.Getenv("INFLUXDB_ORG")
	bucket := os.Getenv("INFLUXDB_BUCKET")

	// Check if any required variable is missing
	if url == "" || token == "" || org == "" || bucket == "" {
		log.Fatal("Missing one or more required environment variables")
	}

	// Create a new InfluxDB client
	client := influxdb2.NewClient(url, token)
	defer client.Close()

	// Create a write API
	writeAPI := client.WriteAPIBlocking(org, bucket)

	// Step 1: Create a new KiezboxStatus message
	statusData := &status.KiezboxStatus{
		BoxId:            1,
		DistId:           101,
		RouterPowered:    true,
		UnixTime:         time.Now().Unix(),
		TemperatureOut:   25000,  // 25°C in milli Celsius
		TemperatureIn:    22000,  // 22°C in milli Celsius
		HumidityIn:       50000,  // 50% in milli Percentage
		SolarVoltage:     12000,  // 12V in milli Volts
		SolarPower:       150,    // 1.5W in deci Watts
		SolarEnergyDay:   5000,   // 50 dW (0.5W) collected today
		SolarEnergyTotal: 100000, // 1000 dW (10W) collected total
		BatteryVoltage:   3700,   // 3.7V in milli Volts
		BatteryCurrent:   500,    // 500mA
		TemperatureRtc:   20000,  // 20°C in milli Celsius
	}

	// Marshal the SensorData message
	marshalledData, err := status.MarshalKiezboxStatus(statusData)
	if err != nil {
		log.Fatalf("Error marshalling data: %v", err)
	}

	// Display the marshalled data
	fmt.Printf("Marshalled Data: %x\n", marshalledData)

	// Unmarshal the data back into a SensorData message
	unmarshalledData, err := status.UnmarshalKiezboxStatus(marshalledData)
	if err != nil {
		log.Fatalf("Error unmarshalling data: %v", err)
	}

	// Prepare the InfluxDB point
	point := influxdb2.NewPoint(
		// Measurement
		"sensor_data",
		// Tags
		map[string]string{
			"box_id":   fmt.Sprintf("%d", unmarshalledData.BoxId),
			"district": fmt.Sprintf("%d", unmarshalledData.DistId),
		},
		// Fields
		map[string]interface{}{
			"router_powered":     unmarshalledData.RouterPowered,
			"temperature_out":    float32(unmarshalledData.TemperatureOut) / 1000, // Converting to float32 and °C
			"temperature_in":     float32(unmarshalledData.TemperatureIn) / 1000,
			"humidity_in":        float32(unmarshalledData.HumidityIn) / 1000,
			"solar_voltage":      float32(unmarshalledData.SolarVoltage) / 1000,
			"solar_power":        float32(unmarshalledData.SolarPower) / 100,
			"solar_energy_day":   float32(unmarshalledData.SolarEnergyDay) / 100,
			"solar_energy_total": float32(unmarshalledData.SolarEnergyTotal) / 100,
			"battery_voltage":    float32(unmarshalledData.BatteryVoltage) / 1000,
			"battery_current":    unmarshalledData.BatteryCurrent,
			"rtc_temperature":    float32(unmarshalledData.TemperatureRtc) / 1000,
		},
		// Timestamp
		time.Unix(unmarshalledData.UnixTime, 0),
	)

	// Write the point to InfluxDB
	if err := writeAPI.WritePoint(context.Background(), point); err != nil {
		log.Fatalf("Failed to write data: %v", err)
	}

	fmt.Println("Data written to InfluxDB successfully")

	// --- Retrieving Data from InfluxDB ---
	// Create a query API
	queryAPI := client.QueryAPI(org)

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
	result, err := queryAPI.Query(context.Background(), query)
	if err != nil {
		log.Fatalf("Error executing query: %v", err)
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
