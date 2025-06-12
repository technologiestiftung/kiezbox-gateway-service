// Package meshtastic provides utility functions for communication with a meshtastic device over serial
package meshtastic

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/tarm/serial"
	"google.golang.org/protobuf/proto"

	"kiezbox/internal/db"
	"kiezbox/internal/github.com/meshtastic/go/generated"
)

// Constants used in the meshtastic stream protocol
// which is documented here > https://meshtastic.org/docs/development/device/client-api/#streaming-version
const (
	start1       = 0x94
	start2       = 0xC3
	maxProtoSize = 512
)

// SerialPort defines the interface for serial port operations.
type SerialPort interface {
	io.ReadWriteCloser
}

type portFactory func(*serial.Config) (SerialPort, error)

// MTSerial represents a connection to a meshtastic device via serial
type MTSerial struct {
	conf          *serial.Config
	port          SerialPort
	config_id     uint32
	ToChan        chan *generated.ToRadio
	FromChan      chan *generated.FromRadio
	KBChan        chan *generated.KiezboxMessage
	ConfigChan    chan *generated.AdminMessage
	MyInfo        *generated.MyNodeInfo
	WaitInfo      sync.WaitGroup
	portFactory   portFactory
	retryTime     int
	apiPort       string
	cacheDir      string
	apiSessionDir string
}

func interfaceIsNil(i interface{}) bool {
	return i == nil || (reflect.ValueOf(i).Kind() == reflect.Ptr && reflect.ValueOf(i).IsNil())
}

// BuildKiezboxControl creates a KiezboxMessage_Control struct with the provided values.
// Pass unixTime as a pointer (or nil) and mode as a pointer (or nil) to set one or both fields.
func BuildKiezboxControl(unixTime *int64, mode *int32) *generated.KiezboxMessage_Control {
	control := &generated.KiezboxMessage_Control{}

	if unixTime != nil {
		control.Set = &generated.KiezboxMessage_Control_UnixTime{
			UnixTime: *unixTime,
		}
	}
	if mode != nil {
		control.Set = &generated.KiezboxMessage_Control_Mode{
			Mode: generated.KiezboxMessage_Mode(*mode),
		}
	}
	return control
}

// Init initializes the serial device of an MTSerial object
// and also sends the necessary initial radioConfig protobuf packet
// to start the communication with the meshtastic serial device
func (mts *MTSerial) Init(dev string, baud int, retryTime int, apiPort string, portFactory portFactory, cacheDir string, api_sessiondir string) {
	mts.FromChan = make(chan *generated.FromRadio, 10)
	mts.ToChan = make(chan *generated.ToRadio, 10)
	mts.KBChan = make(chan *generated.KiezboxMessage, 10)
	mts.ConfigChan = make(chan *generated.AdminMessage, 10)
	mts.WaitInfo.Add(1)
	mts.config_id = rand.Uint32()
	mts.conf = &serial.Config{
		Name: dev,
		Baud: baud,
	}
	mts.portFactory = portFactory
	mts.retryTime = retryTime
	mts.apiPort = apiPort
	mts.cacheDir = cacheDir
	mts.apiSessionDir = api_sessiondir
	var err = mts.Open()
	if err != nil {
		slog.Info("Serial port not available yet. Reader will retry opening it.")
	}
}

// Opens the serial port and sends the necessary initial radioConfig protobuf packet
// to start the communication with the meshtastic serial device
func (mts *MTSerial) Open() (err error) {
	mts.port, err = mts.portFactory(mts.conf)
	if err != nil {
		slog.Error("Failed to open serial port", "err", err)
		return err
	}
	mts.WantConfig()
	return nil
}

func (mts *MTSerial) WantConfig() {
	slog.Info("Serial port opened successfully", "baud", mts.conf.Baud)
	radioConfig := &generated.ToRadio{
		PayloadVariant: &generated.ToRadio_WantConfigId{
			WantConfigId: mts.config_id,
		},
	}
	slog.Info("Sending ToRadio message", "message", radioConfig)
	mts.Write(radioConfig)
}

// Closes the serial connection and blocks WaitInfo again
func (mts *MTSerial) Close() {
	mts.WaitInfo.Add(1)
	var err error
	err = mts.port.Close()
	if err != nil {
		slog.Error("Failed to close serial port", "err", err)
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
			slog.Info("Heartbeat stopped")
			return
		case t := <-ticker.C:
			Heartbeat := &generated.ToRadio{
				PayloadVariant: &generated.ToRadio_Heartbeat{},
			}
			slog.Info("Sending Heartbeat", "time", t)
			mts.Write(Heartbeat)
		}
	}
}

// SetKiezboxValues sends a Kiezbox control message to the meshtastic device to set values such as time and mode.
// Pass a pointer to generated.KiezboxMessage_Control with the desired fields set.
func (mts *MTSerial) SetKiezboxValues(ctx context.Context, wg *sync.WaitGroup, control *generated.KiezboxMessage_Control) {
	mts.WaitInfo.Wait()
	defer wg.Done()

	// Create the Kiezbox message with the provided control fields
	kiezboxMessage := &generated.KiezboxMessage{
		Control: control,
	}

	// Marshal the Kiezbox message
	kiezboxData, err := proto.Marshal(kiezboxMessage)
	if err != nil {
		log.Printf("Failed to marshal KiezboxMessage: %v", err)
	}

	// Create the Data message
	dataMessage := &generated.Data{
		Portnum: generated.PortNum_KIEZBOX_CONTROL_APP,
		Payload: kiezboxData,
	}

	// Create the MeshPacket
	meshPacket := &generated.MeshPacket{
		From:    0, // TODO: what should be sender id ?
		To:      mts.MyInfo.MyNodeNum,
		Channel: 2, // TODO: get Channel dynamically
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

	log.Printf("Setting Kiezbox values: %+v\n", control)

	// Check if the context has been canceled before attempting to write
	select {
	case <-ctx.Done():
		return
	default:
		mts.Write(toRadio)
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
		slog.Error("Failed to marshal KiezboxMessage", "err", err)
	}

	// Create the Data message
	dataMessage := &generated.Data{
		Portnum: generated.PortNum_KIEZBOX_CONTROL_APP, // Replace with the appropriate port number
		Payload: kiezboxData,
	}

	// Create the MeshPacket
	meshPacket := &generated.MeshPacket{
		From:    0, //TODO: what should be sender id ?
		To:      mts.MyInfo.MyNodeNum,
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

	slog.Info("Setting time", "unix_time", time)

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

	for {
		select {
		case <-ctx.Done():
			// Context has been cancelled, exit the loop
			slog.Info("Writer stopped")
			return
		case ToRadio, ok := <-mts.ToChan:
			if !ok {
				// Channel has been closed, exit the loop
				slog.Info("ToChan closed")
				return
			}
			slog.Info("Sending Protobuf to device", "message", ToRadio)
			pb_marshalled, err := proto.Marshal(ToRadio)
			if err != nil {
				slog.Error("Failed to marshal ToRadio", "err", err)
			}
			hex := fmt.Sprintf("%x", pb_marshalled)
			slog.Info("ToRadio Marshalled", "hex", hex)
			configLen := len(pb_marshalled)
			configHeader := []byte{
				start1,
				start2,
				byte((configLen >> 8) & 0xFF),
				byte(configLen & 0xFF),
			}
			packet := append(configHeader, pb_marshalled...)
			// Debug output
			slog.Info("Sending packet", "hex", fmt.Sprintf("%x", packet))
			// Write the packet to the serial port
			if !interfaceIsNil(mts.port) {
				_, err = mts.port.Write(packet)
				if err != nil {
					slog.Error("Failed to write to serial port", "err", err)
				}
			} else {
				slog.Error("Failed to write data to serial, port is not available")
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

	for {
		select {
		case <-ctx.Done():
			slog.Info("Reader stopped")
			return
		default:
			// Read one byte at a time from the serial port.
			byteBuf := make([]byte, 1)
			var portbroken bool = false
			if interfaceIsNil(mts.port) {
				slog.Error("Serial port is not initialized", "port", mts.port)
				portbroken = true
			} else {
				_, err := mts.port.Read(byteBuf)
				if err != nil {
					slog.Error("Error reading from serial port", "err", err)
					portbroken = true
					mts.Close()
				}
			}
			if portbroken {
				for {
					slog.Info("Waiting for device to reconnect...")
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
					// log.Printf("DEBUG (ASCII): %s\n", ascii)
					// log.Printf("Debug output (Hex): %s\n", hex)
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
					slog.Error("Invalid packet: length exceeds 512 bytes. Ignoring...")
					buffer.Reset() // Reset and continue looking for START1.
					continue
				}

				// Wait until we have the entire protobuf payload.
				if buffer.Len() >= int(4+protoLen) {
					protobufMsg := buffer.Bytes()[4 : 4+protoLen]

					// Log Protobuf frame details for debugging.
					slog.Info("Protobuf frame detected", "length", protoLen)
					slog.Info("Protobuf frame", "hex", fmt.Sprintf("%x", protobufMsg))
					var fromRadio generated.FromRadio
					err := proto.Unmarshal(protobufMsg, &fromRadio)
					if err != nil {
						slog.Error("Failed to unmarshal fromRadio", "err", err)
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
			slog.Info("DBWriter context canceled, shutting down.")
			return
		case message := <-mts.KBChan:
			if message == nil {
				continue
			}

			// Set the arrival time to the current time
			message.Update.ArrivalTime = proto.Int64(time.Now().Unix())

			// Check connection to database before trying to write the point
			databaseConnected, err := db_client.Client.Ping(ctx)
			if !databaseConnected {
				// Cache the message if database is not connected
				slog.Warn("No database connection. Caching point.", "err", err)
				db.WritePointToFile(message, mts.cacheDir)
				continue
			}

			slog.Info("Handling Protobuf message")
			// Convert the Protobuf message to an InfluxDB point
			point, err := db.KiezboxMessageToPoint(message)
			slog.Info("Adding point", "point", point)

			// Write the point to InfluxDB
			err = db_client.WritePointToDatabase(point)

			// Cache message if connection to database failed
			if err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					slog.Warn("No connection to database, caching point.")
					db.WritePointToFile(message, mts.cacheDir)

				} else {
					slog.Error("Unexpected error", "err", err)
				}
			} else {
				slog.Info("Data written to InfluxDB successfully")
			}
		}
	}
}

// DBRetry tries to write cached points to the InfluxDB instance.
func (mts *MTSerial) DBRetry(ctx context.Context, wg *sync.WaitGroup, db_client *db.InfluxDB) {
	// Decrement WaitGroup when function exits
	defer wg.Done()

	// Do retry every mts.retryTime seconds
	ticker := time.NewTicker(time.Duration(mts.retryTime) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Retry goroutine shutting down.")
			return
		case <-ticker.C:
			// Check if the database is connected before retrying
			databaseConnected, err := db_client.Client.Ping(ctx)

			if databaseConnected {
				slog.Info("Database connected, retrying cached points.")
				db_client.RetryCachedPoints(mts.cacheDir)

			} else {
				slog.Warn("No database connection. Skipping retry.", "err", err)
			}
		}
	}
}

// GetConfig sends a periodic request to the meshtastic device to get the current configuration
func (mts *MTSerial) GetConfig(ctx context.Context, wg *sync.WaitGroup, interval time.Duration) {
	mts.WaitInfo.Wait()
	// Decrement WaitGroup when function exits
	defer wg.Done()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("GetConfig stopped")
			return
		case <-ticker.C:
			// Create the Admin message
			adminMessage := &generated.AdminMessage{
				PayloadVariant: &generated.AdminMessage_GetModuleConfigRequest{
					GetModuleConfigRequest: generated.AdminMessage_KIEZBOXCONTROL_CONFIG,
				},
			}

			// Marshal the Admin message
			adminData, err := proto.Marshal(adminMessage)
			if err != nil {
				fmt.Printf("Failed to marshal AdminMessage: %v", err)
			}

			// Create the Data message
			dataMessage := &generated.Data{
				Portnum: generated.PortNum_KIEZBOX_CONTROL_APP, // Replace with the appropriate port number
				Payload: adminData,
			}

			// Create the MeshPacket
			meshPacket := &generated.MeshPacket{
				From:    0, //TODO: what should be sender id ?
				To:      mts.MyInfo.MyNodeNum,
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

			// Send the message
			mts.Write(toRadio)

			fmt.Printf("Sending config request")
		}
	}
}

// ConfigWriter saves the current configuration of the meshtastic device
func (mts *MTSerial) ConfigWriter(ctx context.Context, wg *sync.WaitGroup) {
	// Decrement WaitGroup when function exits
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			// Exit gracefully when the context is canceled
			fmt.Println("ConfigWriter context canceled, shutting down.")
			return
		case message := <-mts.ConfigChan:
			if message == nil {
				continue
			}
			config := message.GetGetModuleConfigResponse()
			if config != nil {
				kiezboxControl := config.GetKiezboxControl()
				if kiezboxControl != nil {
					mode := kiezboxControl.GetMode()
					// TODO: Get rid of print statements and save value so it can be retrieved by the API
					fmt.Printf("Current mode: %d\n", mode)
				} else {
					fmt.Println("Mode is nil")
				}
			}
		}
	}
}

// APIHandler starts the API for the Kiezbox Gateway Service.
func (mts *MTSerial) APIHandler(ctx context.Context, wg *sync.WaitGroup, r *gin.Engine) {
	// Decrement WaitGroup when function exits
	defer wg.Done()

	// Configure the HTTP server
	server := &http.Server{
		Addr:    fmt.Sprintf("localhost:%s", mts.apiPort),
		Handler: r,
	}

	// Start the HTTP server
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Failed to start API server", "err", err)
	}

	// Handle context cancellation and server shutdown
	<-ctx.Done()
	slog.Info("Shutting down API server...")
	if err := server.Shutdown(ctx); err != nil {
		slog.Error("API server forced to shut down", "err", err)
	}
}
