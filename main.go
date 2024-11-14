package main

import (
	"fmt"
	"log"
	"time"

	"kiezbox/status"
)

func main() {
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

	// Step 4: Display the unmarshalled data
	fmt.Println("\nUnmarshalled Data:")
	fmt.Printf("Box ID: %d\n", unmarshalledData.BoxId)
	fmt.Printf("District ID: %d\n", unmarshalledData.DistId)
	fmt.Printf("Router Powered: %v\n", unmarshalledData.RouterPowered)
	fmt.Printf("Unix Time: %s\n", time.Unix(unmarshalledData.UnixTime, 0).Format(time.RFC3339))
	fmt.Printf("Temperature Outside: %.2f°C\n", float32(unmarshalledData.TemperatureOut)/1000)
	fmt.Printf("Temperature Inside: %.2f°C\n", float32(unmarshalledData.TemperatureIn)/1000)
	fmt.Printf("Humidity Inside: %.2f%%\n", float32(unmarshalledData.HumidityIn)/1000)
	fmt.Printf("Solar Voltage: %.2fV\n", float32(unmarshalledData.SolarVoltage)/1000)
	fmt.Printf("Solar Power: %.1fW\n", float32(unmarshalledData.SolarPower)/100)
	fmt.Printf("Solar Energy Today: %.1fW\n", float32(unmarshalledData.SolarEnergyDay)/100)
	fmt.Printf("Solar Energy Total: %.1fW\n", float32(unmarshalledData.SolarEnergyTotal)/100)
	fmt.Printf("Battery Voltage: %.2fV\n", float32(unmarshalledData.BatteryVoltage)/1000)
	fmt.Printf("Battery Current: %dmA\n", unmarshalledData.BatteryCurrent)
	fmt.Printf("RTC Temperature: %.2f°C\n", float32(unmarshalledData.TemperatureRtc)/1000)
}
