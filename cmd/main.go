package main

import (
	"flag"
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"google.golang.org/protobuf/reflect/protoreflect"
	"kiezbox/internal/config"
	"kiezbox/internal/db"
	"kiezbox/internal/meshtastic"
	"log"
	"os"
	"time"
)

func main() {
	flag_settime := flag.Bool("settime", false, "Sets the RTC time to the system time at service startup")
	flag_daemon := flag.Bool("daemon", false, "Tells the serice to run as (background) daemon")
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

	// Process Protobuf messages in the main goroutine.
	//TODO: move this into it's own gorouting
	if *flag_daemon {
		for message := range mts.KBChan {
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
	} else {
		mts.WaitInfo.Wait()
	}

	// --- Retrieve Data from InfluxDB ---
	// Flux query to get latest measurements
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
