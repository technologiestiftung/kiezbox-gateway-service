package main

import (
	"fmt"
	"log"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"

	config "kiezbox/internal/config"
	db "kiezbox/internal/db"
	"kiezbox/internal/github.com/meshtastic/go/generated"
	"bytes"
	"encoding/binary"
        "github.com/tarm/serial"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/encoding/prototext"
        "google.golang.org/protobuf/reflect/protoreflect"
)

const (
	start1       = 0x94
	start2       = 0xC3
	maxProtoSize = 512
)

func main() {
	// --- Mock Kiezbox data, marshalling and unmarshalling ---
    	// Configure the serial port with baud rate and other settings.
	serialconfig := &serial.Config{
		Name: "/dev/ttyUSB0", // Replace with your serial port
		Baud: 115200,         // Set the desired baud rate here
	}
	port, err := serial.OpenPort(serialconfig)
	if err != nil {
		log.Fatalf("Failed to open serial port: %v", err)
	}
	defer port.Close()

	fmt.Println("Serial port opened successfully with baud rate:", serialconfig.Baud)

	// Create channels for handling Protobuf messages.
	protoChan := make(chan *generated.KiezboxMessage)

	// Launch a goroutine for serial reading.
	go readSerial(port, protoChan)

	// --- Write data to the DB
	// Load InfluxDB configuration
	url, token, org, bucket := config.LoadConfig()

	// Initialize InfluxDB client
	db_client := db.CreateClient(url, token, org, bucket)
	defer db_client.Close()

	// Process Protobuf messages in the main goroutine.
        //TODO: move this into it's own gorouting
	for message := range protoChan {
		fmt.Println("Handling Protobuf message")
                debugPrintProtobuf(message)
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
                        // Get the fields
                        fields[string(fd.Name())] = v.Interface()
                        return true // Continue iteration
                })
                // Prepare the InfluxDB point
                point := influxdb2.NewPoint(
                        // Measurement
                        "core_values",
                        // Tags
                        tags,
                        // Fields
                        fields,
                        // Timestamp
                        //time.Unix(message.Update.UnixTime, 0),
                        time.Now(),
                )
                fmt.Printf("Addint point: %+v\n", point)

                // Write the point to InfluxDB
                err := db_client.WriteData(point)
                fmt.Println("Writing data:", err)

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

func readSerial(port *serial.Port, protoChan chan<- *generated.KiezboxMessage) {
	var buffer bytes.Buffer
	var debugBuffer bytes.Buffer

        radioConfig := &generated.ToRadio{
            PayloadVariant: &generated.ToRadio_WantConfigId{
                WantConfigId: 32,
            },
        }
        fmt.Println("radioConfig:", radioConfig)
        fmt.Printf("ToRadio message: %+v\n", radioConfig)
        configMarshalled, err := proto.Marshal(radioConfig)
	if err != nil {
		fmt.Println("failed to marshal SensorData: %w", err)
	}
        hex := fmt.Sprintf("%x", configMarshalled)
        fmt.Printf("radioMarshalled: 0x%s\n", hex)
        configLen := len(configMarshalled)
        configHeader := []byte{
            start1,
            start2,
            byte((configLen >> 8) & 0xFF),
            byte(configLen & 0xFF),
        }
        packet := append(configHeader, configMarshalled...)

	// Debug output
	log.Printf("Sending packet (Hex): %x\n", packet)

	// Write the packet to the serial port
	_, err = port.Write(packet)
	if err != nil {
		fmt.Println("failed to write to serial port: %w", err)
	}

	for {
		// Read one byte at a time from the serial port.
		byteBuf := make([]byte, 1)
		_, err := port.Read(byteBuf)
		if err != nil {
			log.Printf("Error reading from serial port: %v\n", err)
			return
		}

		b := byteBuf[0]

		// Check for START1 and START2 in the buffer.
		if buffer.Len() == 0 && b != start1 {
			// Accumulate bytes for debug output.
			if b == '\n' {
				// Print debug output when a newline is detected.
				// ascii := debugBuffer.String()
				// hex := fmt.Sprintf("%x", debugBuffer.Bytes())
				// fmt.Printf("DEBUG (ASCII): %s\n", ascii)
                                // fmt.Printf("Debug output (Hex): %s\n", hex)
				debugBuffer.Reset()
			} else {
				debugBuffer.WriteByte(b)
			}
			continue
		}

		// Accumulate bytes into the buffer.
		buffer.WriteByte(b)

		// Look for START1 and START2 sequence.
		if buffer.Len() == 2 && !(buffer.Bytes()[0] == start1 && buffer.Bytes()[1] == start2) {
			// Not a valid protobuf start, reset the buffer.
			buffer.Reset()
			continue
		}

		// Once the buffer contains the header, check for the length.
		if buffer.Len() >= 4 {
			header := buffer.Bytes()
			protoLen := binary.BigEndian.Uint16(header[2:4])

			if protoLen > maxProtoSize {
				fmt.Println("Invalid packet: length exceeds 512 bytes. Ignoring...")
				buffer.Reset() // Reset and continue looking for START1.
				continue
			}

			// Wait until we have the entire protobuf payload.
			if buffer.Len() >= int(4+protoLen) {
				protobufMsg := buffer.Bytes()[4 : 4+protoLen]

				// Log Protobuf frame details for debugging.
				fmt.Printf("Protobuf frame detected! Length: %d bytes\n", protoLen)
				fmt.Printf("Protobuf frame (Hex): %x\n", protobufMsg)
                                var fromRadio generated.FromRadio
                                err := proto.Unmarshal(protobufMsg, &fromRadio)
                                if err != nil {
                                        fmt.Println("failed to unmarshal fromRadio: %w", err)
                                }
                                debugPrintProtobuf(&fromRadio)
                                switch v := fromRadio.PayloadVariant.(type) {
                                case *generated.FromRadio_Packet:
                                    debugPrintProtobuf(v.Packet)
                                    switch v := v.Packet.PayloadVariant.(type) {
                                    case *generated.MeshPacket_Decoded:
                                        debugPrintProtobuf(v.Decoded)
                                        switch v.Decoded.Portnum {
                                        case generated.PortNum_KIEZBOX_CONTROL_APP:
                                            var KiezboxMessage generated.KiezboxMessage
                                            err := proto.Unmarshal(v.Decoded.Payload, &KiezboxMessage)
                                            if err != nil {
                                                    fmt.Println("failed to unmarshal KiezboxMessage: %w", err)
                                            }
                                            debugPrintProtobuf(&KiezboxMessage)
                                            // Send the protobuf message to the processing goroutine.
                                            protoChan <- &KiezboxMessage
                                        default:
                                            fmt.Println("Payload variant not a Kiezbox Message")
                                        }
                                    default:
                                        fmt.Println("Payload variant is encrypted")
                                    }
                                default:
                                    fmt.Println("Payload variant is not 'packet'")
                                }
				// Remove the processed message from the buffer.
				buffer.Reset()
			}
		}
	}
}

func debugPrintProtobuf(message proto.Message) {
	// Convert the Protobuf message to text format
	textData, err := prototext.MarshalOptions{
		Multiline: true, // Use multiline output for readability
	}.Marshal(message)

	if err != nil {
		log.Printf("Failed to marshal Protobuf message to text: %v", err)
		return
	}

	// Print the formatted Protobuf message
	fmt.Println("Protobuf message content (Text):")
	fmt.Println(string(textData))
}

// TODO: debug `DEBUG (ASCII): INFO  | ??:??:?? 6439 [SerialConsole] Lost phone connection`
// Defaulting to the formerly removed phone_timeout_secs value of 15 minutes
// #define SERIAL_CONNECTION_TIMEOUT (15 * 60) * 1000UL
