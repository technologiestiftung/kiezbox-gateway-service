package testutils

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	influxdb "github.com/influxdata/influxdb-client-go/v2"
	influxdb_write "github.com/influxdata/influxdb-client-go/v2/api/write"
	"google.golang.org/protobuf/proto"

	"kiezbox/internal/github.com/meshtastic/go/generated"
)

// Create a basic KiezboxMessage
func CreateKiezboxMessage(timestamp int64) *generated.KiezboxMessage {
	return &generated.KiezboxMessage{
		Update: &generated.KiezboxMessage_Update{
			Meta: &generated.KiezboxMessage_Meta{
				BoxId:  proto.Uint32(1),
				DistId: proto.Uint32(2),
			},
			UnixTime: timestamp,
			Core: &generated.KiezboxMessage_Core{
				Mode: generated.KiezboxMessage_normal,
				Router: &generated.KiezboxMessage_Router{
					Powered: true,
				},
				Values: &generated.KiezboxMessage_CoreValues{},
			},
		},
	}
}

// Create a file with a marshaled KiezboxMessage
func CreateKiezboxMessageFile(dir string) {
	// Create the message
	message := CreateKiezboxMessage(time.Now().Unix())

	// Set the arrival time to the current time
	message.Update.ArrivalTime = proto.Int64(time.Now().Unix())

	// Marshal the message
	marshaled, err := proto.Marshal(message)
	if err != nil {
		fmt.Println("Error marshaling message:", err)
		return
	}

	// Generate the filename
	filename := fmt.Sprintf("%s.pb", uuid.New().String())

	// Create the full file path
	filePath := filepath.Join(".", filename)

	// Write the marshaled data to the file
	err = os.WriteFile(filePath, marshaled, 0666)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}

	fmt.Println("File saved successfully:", filePath)
}

// Create a pre-defined InfluxDB Point
func CreateTestPoint() *influxdb_write.Point {
	return influxdb.NewPoint(
		// Measurement
		"sensor_data",
		// Tags
		map[string]string{
			"box_id":   "1",
			"district": "1",
		},
		// Fields
		map[string]any{
			"router_powered":  true,
			"temperature_out": float32(30),
			"temperature_in":  float32(28),
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
			"box_id":   "1",
			"district": "1",
		},
		// Fields
		map[string]any{
			"router_powered":  true,
			"temperature_out": float32(30),
			"temperature_in":  float32(28),
		},
		// Timestamp
		time.Unix(time.Now().Unix(), 0),
	)
}

// Create a valid InfluxDB query string
func CreateQuery(bucket string) string {
	return `
		from(bucket: "` + bucket + `")
			|> range(start: -1h)
			|> filter(fn: (r) => r["_measurement"] == "sensor_data")
			|> filter(fn: (r) => r["_field"] == "temperature_out" or r["_field"] == "temperature_in" or r["_field"] == "humidity_in")
			|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
			|> yield(name: "sensor_data")
	`
}
