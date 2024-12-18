package main

import (
	"context"
	"flag"
	"fmt"
	"kiezbox/internal/config"
	"kiezbox/internal/db"
	"kiezbox/internal/meshtastic"
	"os"
	"sync"
	"time"

	"github.com/tarm/serial"
)

// Using an interface as an intermediate layer instead of calling the meshtastic functions directly
// allows for dependency injection, essential for unittesting.
type MeshtasticDevice interface {
    Writer(ctx context.Context, wg *sync.WaitGroup)
    Heartbeat(ctx context.Context, wg *sync.WaitGroup, interval time.Duration)
	Reader(ctx context.Context, wg *sync.WaitGroup)
	MessageHandler(ctx context.Context, wg *sync.WaitGroup)
	DBWriter(ctx context.Context, wg *sync.WaitGroup, db_client *db.InfluxDB)
	Settime(ctx context.Context, wg *sync.WaitGroup, time int64)
}


// RunGoroutines orchestrates the goroutines that run the service.
func RunGoroutines(ctx context.Context, wg *sync.WaitGroup, device MeshtasticDevice, setTime bool, daemon bool, db_client *db.InfluxDB) {
	// Launch goroutines
	wg.Add(1)
	go device.Writer(ctx, wg)
	wg.Add(1)
	go device.Heartbeat(ctx, wg, 30 * time.Second)
	wg.Add(1)
	go device.Reader(ctx, wg)
	wg.Add(1)
	go device.MessageHandler(ctx, wg)
	if setTime {
		// We wait for the not info to set the time
		wg.Add(1)
		go device.Settime(ctx, wg, time.Now().Unix())
	}

	// Process incoming KiexBox messages in its own goroutine
	if daemon {
		wg.Add(1)
		go device.DBWriter(ctx, wg, db_client)
	// } else {
	// 	mts.WaitInfo.Wait()
	}
}

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

	portFactory := func(conf *serial.Config) (meshtastic.SerialPort, error) {
		return serial.OpenPort(conf)
}
	mts.Init("/dev/ttyUSB0", 115200, portFactory)

	// Load InfluxDB configuration
	url, token, org, bucket := config.LoadConfig()

	// Initialize InfluxDB client
	db_client := db.CreateClient(url, token, org, bucket)
	defer db_client.Close()

	// Create a context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a WaitGroup to wait for the goroutines
	var wg sync.WaitGroup

	// Run the goroutines
	RunGoroutines(ctx, &wg, &mts, *flag_settime, *flag_daemon, db_client)

    // Wait for all goroutines to finish
    wg.Wait()
}
