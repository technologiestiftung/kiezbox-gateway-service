// Package meshtastic provides utility functions for communication with a meshtastic device over serial
package meshtastic

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/tarm/serial"
	"google.golang.org/protobuf/proto"

	"kiezbox/internal/db"
	"kiezbox/internal/github.com/meshtastic/go/generated"
)

// Constants used in the meshtastic stream protocol
// which is documented here > https://meshtastic.org/docs/development/device/client-api/#streaming-version
const (
	start1	     = 0x94
	start2	     = 0xC3
	maxProtoSize = 512
)

// SerialPort defines the interface for serial port operations.
type SerialPort interface {
	io.ReadWriteCloser
}

type portFactory func(*serial.Config) (SerialPort, error)

// MTSerial represents a connection to a meshtastic device via serial
type MTSerial struct {
	conf      *serial.Config
	port      SerialPort
	config_id uint32
	ToChan	  chan *generated.ToRadio
	FromChan  chan *generated.FromRadio
	KBChan	  chan *generated.KiezboxMessage
	MyInfo	  *generated.MyNodeInfo
	WaitInfo  sync.WaitGroup
	portFactory portFactory
	retryTime int
}

// Init initializes the serial device of an MTSerial object
// and also sends the necessary initial radioConfig protobuf packet
// to start the communication with the meshtastic serial device
func (mts *MTSerial) Init(dev string, baud int, retryTime int, portFactory portFactory) {
	mts.FromChan = make(chan *generated.FromRadio, 10)
	mts.ToChan = make(chan *generated.ToRadio, 10)
	mts.KBChan = make(chan *generated.KiezboxMessage, 10)
	mts.WaitInfo.Add(1)
	mts.config_id = rand.Uint32()
	mts.conf = &serial.Config{
		Name: dev,
		Baud: baud,
	}
	mts.portFactory = portFactory
	mts.retryTime = retryTime
	for {
		var err = mts.Open()
		if err == nil {
			break
		} else {
		    fmt.Println("Waiting for serial Port to become available...")
		    time.Sleep(time.Second * 3)
		}
	}
}

// Opens the serial port and sends the necessary initial radioConfig protobuf packet
// to start the communication with the meshtastic serial device
func (mts *MTSerial) Open() (err error) {
	mts.port, err = mts.portFactory(mts.conf)
	if err != nil {
		fmt.Printf("Failed to open serial port: %v\n", err)
		return err;
	}
	mts.WantConfig()
	return nil;
}

func (mts *MTSerial) WantConfig() {
	fmt.Println("Serial port opened successfully with baud rate:", mts.conf.Baud)
	radioConfig := &generated.ToRadio{
		PayloadVariant: &generated.ToRadio_WantConfigId{
			WantConfigId: mts.config_id,
		},
	}
	fmt.Printf("Sending ToRadio message: %+v\n", radioConfig)
	mts.Write(radioConfig)
}

// Closes the serial connection and blocks WaitInfo again
func (mts *MTSerial) Close() {
	mts.WaitInfo.Add(1)
	var err error
	err = mts.port.Close()
	if err != nil {
		fmt.Printf("Failed to close serial port: %v", err)
	}
}

// Heartbeat sends a periodic heartbeat message to the meshtastic device to keep the serial connection alive
func (mts *MTSerial) Heartbeat(ctx context.Context, wg *sync.WaitGroup, interval time.Duration) {
	// Decrement WaitGroup when function exits
	defer wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Heartbeat stopped")
			return
		case t := <-ticker.C:
		Heartbeat := &generated.ToRadio{
			PayloadVariant: &generated.ToRadio_Heartbeat{},
		}
		fmt.Printf("Sending Heartbeat at %s\n", t)
		mts.Write(Heartbeat)
		}
	}
}

// Settime sends a Kiezbox control message to the meshtastic device containing the current system time
// The meshtastic device uses it to update its own RTC to the new value
func (mts *MTSerial) Settime(ctx context.Context, wg *sync.WaitGroup, time int64) {
	mts.WaitInfo.Wait()
	// Decrement WaitGroup when function exits
	defer wg.Done()

	// Create the Kiezbox message
	kiezboxMessage := &generated.KiezboxMessage{
		Control: &generated.KiezboxMessage_Control{
			Set: &generated.KiezboxMessage_Control_UnixTime{
				UnixTime: time,
			},
		},
	}

	// Marshal the Kiezbox message
	kiezboxData, err := proto.Marshal(kiezboxMessage)
	if err != nil {
		fmt.Printf("Failed to marshal KiezboxMessage: %v", err)
	}

	// Create the Data message
	dataMessage := &generated.Data{
		Portnum: generated.PortNum_KIEZBOX_CONTROL_APP, // Replace with the appropriate port number
		Payload: kiezboxData,
	}

	// Create the MeshPacket
	meshPacket := &generated.MeshPacket{
		From:	 0, //TODO: what should be sender id ?
		To:	 mts.MyInfo.MyNodeNum,
		Channel: 2, //TODO: get Channel dynamically
		PayloadVariant: &generated.MeshPacket_Decoded{
			Decoded: dataMessage,
		},
	}

	// Create the ToRadio message
	toRadio := &generated.ToRadio{
		PayloadVariant: &generated.ToRadio_Packet{
			Packet: meshPacket,
		},
	}


	fmt.Printf("Setting time to unix time %d\n", time)

    // Check if the context has been canceled before attempting to write
    select {
    case <-ctx.Done():
        return
    default:
        // Send the message
        mts.Write(toRadio)
    }

	return
}

// Write takes a ToRadio protobuf and writes it to the ToChan to be processed by the Writer
func (mts *MTSerial) Write(toradio *generated.ToRadio) {
	mts.ToChan <- toradio
}

// Writer takes a ToRadio protobuf from to ToChan, marshalls it and sends it over the serial
// connection to the meshtastic device. The necessary framing is done here.
func (mts *MTSerial) Writer(ctx context.Context, wg *sync.WaitGroup) {
	// Decrement WaitGroup when function exits
	defer wg.Done()

	if mts.port == nil {
		fmt.Println("Serial port not initialized")
		return
	}
	for {
		select {
		case <-ctx.Done():
			// Context has been cancelled, exit the loop
			fmt.Println("Writer stopped")
			return
		case ToRadio, ok := <-mts.ToChan:
			if !ok {
				// Channel has been closed, exit the loop
				fmt.Println("ToChan closed")
				return
			}
			fmt.Printf("Sending Protobuf to device: %+v\n", ToRadio)
			pb_marshalled, err := proto.Marshal(ToRadio)
			if err != nil {
				fmt.Println("failed to marshal ToRadio: %w", err)
			}
			hex := fmt.Sprintf("%x", pb_marshalled)
			fmt.Printf("ToRadio Marshalled: 0x%s\n", hex)
			configLen := len(pb_marshalled)
			configHeader := []byte{
				start1,
				start2,
				byte((configLen >> 8) & 0xFF),
				byte(configLen & 0xFF),
			}
			packet := append(configHeader, pb_marshalled...)
			// Debug output
			fmt.Printf("Sending packet (Hex): %x\n", packet)
			// Write the packet to the serial port
			if mts.port != nil {
				_, err = mts.port.Write(packet)
				if err != nil {
					fmt.Println("failed to write to serial port: %w", err)
				}
			} else {
				fmt.Println("failed to write data to serial, as port is not available")
			}
		}
	}
}

// Reader takes a channel to write FromRadio protobuf messages to as they arrive on the serial interface
// It parses the framing information and discards any 'non protobuf' messages that may arrive
// It should probably be started as goroutine, as it never returns and blocks while reading from serial
func (mts *MTSerial) Reader(ctx context.Context, wg *sync.WaitGroup) {
	// Decrement WaitGroup when function exits
	defer wg.Done()

	var buffer bytes.Buffer
	var debugBuffer bytes.Buffer

	if mts.port == nil {
		fmt.Println("Serial port not initialized")
		return
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Reader stopped")
			return
		default:
			// Read one byte at a time from the serial port.
			byteBuf := make([]byte, 1)
			_, err := mts.port.Read(byteBuf)
			if err != nil {
				fmt.Printf("Error reading from serial port: %v\n", err)
				mts.Close()
				for {
					fmt.Println("Waiting for device to reconnect...")
					var err = mts.Open()
					if err == nil {
						break
					}
					time.Sleep(time.Second * 3)
				}	
				continue
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
					} else {
						mts.FromChan <- &fromRadio
					}
					// Remove the processed message from the buffer.
					buffer.Reset()
				}
			}
		}
	}
}

// DBWriter writes the received data to the InfluxDB instance.
func (mts *MTSerial) DBWriter(ctx context.Context, wg *sync.WaitGroup, db_client *db.InfluxDB) {
	// Decrement WaitGroup when function exits
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			// Exit gracefully when the context is canceled
			fmt.Println("DBWriter context canceled, shutting down.")
			return
		case message := <-mts.KBChan:
			if message == nil {
				continue
			}

			// Check connection to database before trying to write the point
			databaseConnected, err := db_client.Client.Ping(ctx)
			if !databaseConnected {
				// Cache the message if database is not connected
				fmt.Println("No database connection. Caching point.", err)
				db.WritePointToFile(message, db.CacheDir)
				return
			}

			fmt.Println("Handling Protobuf message")
			// Convert the Protobuf message to an InfluxDB point
			point, err := db.KiezboxMessageToPoint(message)
			fmt.Printf("Adding point: %+v\n", point)

			// Write the point to InfluxDB
			err = db_client.WritePointToDatabase(point)

			// Cache message if connection to database failed
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					log.Println("No connection to database, caching point.")
					db.WritePointToFile(message, db.CacheDir)
				} else {
					log.Println("Unexpected error:", err)
				}
			} else {
				fmt.Println("Data written to InfluxDB successfully")
			}
		}
	}
}

// DBWriterRetry tries to write cached points to the InfluxDB instance.
func (mts *MTSerial) DBWriterRetry(ctx context.Context, wg *sync.WaitGroup, db_client *db.InfluxDB) {
	// Decrement WaitGroup when function exits
	defer wg.Done()

	// Do retry every 10 seconds
	ticker := time.NewTicker(time.Duration(mts.retryTime) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Retry goroutine shutting down.")
			return
		case <-ticker.C:
			// Check if the database is connected before retrying
			databaseConnected, err := db_client.Client.Ping(ctx)

			if databaseConnected {
				fmt.Println("Database connected, retrying cached points.")
				db_client.RetryCachedPoints(db.CacheDir)
			} else {
				fmt.Println("No database connection. Skipping retry.", err)
			}
		}
	}
}
