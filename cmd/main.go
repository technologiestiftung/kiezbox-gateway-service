package main

import (
	"flag"
	"fmt"
	"kiezbox/internal/config"
	"kiezbox/internal/db"
	"kiezbox/internal/meshtastic"
	"log"
	"os"
	"time"
)

func main() {
	flag_settime := flag.Bool("settime", false, "Sets the RTC time to the system time at service startup")
	flag_daemon := flag.Bool("daemon", false, "Tells the service to run as (background) daemon")
	flag_help := flag.Bool("help", false, "Prints the help info and exits")
	flag.Parse()
	// Print help and exit
	if *flag_help {
		fmt.Println("Kiezbox Gateway Service.")
		fmt.Println("Usage flags:")
		flag.VisitAll(func(f *flag.Flag) {
			fmt.Printf("  -%s: %s (default: %s)\n", f.Name, f.Usage, f.DefValue)
		})
		os.Exit(0)
	}

	// Initialize meshtastic serial connection
	var mts meshtastic.MTSerial
	mts.Init("/dev/ttyUSB0", 115200)

	// Launch a goroutine for serial reading.
	go mts.Writer()
	go mts.Heartbeat(30 * time.Second)
	go mts.Reader()
	go mts.MessageHandler()
	if *flag_settime {
		// We wait for the not info to set the time
		go mts.Settime(time.Now().Unix())
	}

	// Load InfluxDB configuration
	url, token, org, bucket := config.LoadConfig()

	// Initialize InfluxDB client
	db_client := db.CreateClient(url, token, org, bucket)
	defer db_client.Close()


	// Process incoming KiexBox messages in its own goroutine
	if *flag_daemon {
		go mts.DBWriter(db_client)
	} else {
		mts.WaitInfo.Wait()
	}

	// --- Retrieve Data from InfluxDB ---
	// Flux query to get latest measurements
	// This is just for testing purposes, but actually not a responsibility of the gateway service
	query := fmt.Sprintf(`
		from(bucket: "%s")
			|> range(start: -15m)  // Retrieve data from the last 1 hour
			|> filter(fn: (r) => r["_measurement"] == "core_values")
	`, bucket)

	// Execute the query
	result, err := db_client.QueryData(query)
	if err != nil {
		log.Fatalf("Error querying data: %v", err)
	}

	// Iterate over the query result and print the data
	for result.Next() {
		fmt.Printf("Time: %v, Measurement: %v, Field: %v, Value: %v\n",
			result.Record().Time(),
			result.Record().Measurement(),
			result.Record().Field(),
			result.Record().Value())
	}

	// Check for errors in the query results
	if result.Err() != nil {
		log.Fatalf("Query failed: %v", result.Err())
	}

	fmt.Println("Data retrieved from InfluxDB successfully")
}
