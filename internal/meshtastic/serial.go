package meshtastic

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/tarm/serial"
	"google.golang.org/protobuf/proto"
	"kiezbox/internal/github.com/meshtastic/go/generated"
	"math/rand"
	"time"
)

const (
	start1       = 0x94
	start2       = 0xC3
	maxProtoSize = 512
)

type MTSerial struct {
	conf      *serial.Config
	port      *serial.Port
	config_id uint32
}

func (mts *MTSerial) Init(dev string, baud int) {
	mts.config_id = rand.Uint32()
	mts.conf = &serial.Config{
		Name: dev,
		Baud: baud,
	}
	var err error
	mts.port, err = serial.OpenPort(mts.conf)
	if err != nil {
		fmt.Print("Failed to open serial port: %v", err)
	} else {
		fmt.Println("Serial port opened successfully with baud rate:", mts.conf.Baud)
		radioConfig := &generated.ToRadio{
			PayloadVariant: &generated.ToRadio_WantConfigId{
				WantConfigId: mts.config_id,
			},
		}
		fmt.Printf("Sending ToRadio message: %+v\n", radioConfig)
		mts.Write(radioConfig)
	}
}

func (mts *MTSerial) Close() {
	mts.port.Close()
}

func (mts *MTSerial) Write(pb *generated.ToRadio) {
	pb_marshalled, err := proto.Marshal(pb)
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
	_, err = mts.port.Write(packet)
	if err != nil {
		fmt.Println("failed to write to serial port: %w", err)
	}
}

func (mts *MTSerial) Read(protoChan chan<- *generated.FromRadio) {
	var buffer bytes.Buffer
	var debugBuffer bytes.Buffer
	for {
		// Read one byte at a time from the serial port.
		byteBuf := make([]byte, 1)
		_, err := mts.port.Read(byteBuf)
		if err != nil {
			fmt.Printf("Error reading from serial port: %v\n", err)
			time.Sleep(time.Second * 3)
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
					protoChan <- &fromRadio
				}
				// Remove the processed message from the buffer.
				buffer.Reset()
			}
		}
	}
}

// TODO: debug `DEBUG (ASCII): INFO  | ??:??:?? 6439 [SerialConsole] Lost phone connection`
// Defaulting to the formerly removed phone_timeout_secs value of 15 minutes
// #define SERIAL_CONNECTION_TIMEOUT (15 * 60) * 1000UL
