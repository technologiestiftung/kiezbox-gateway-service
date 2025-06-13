package main

import (
	"context"
	"flag"
	"fmt"
	"kiezbox/api/routes"
	"kiezbox/internal/config"
	"kiezbox/internal/db"
	"kiezbox/internal/github.com/meshtastic/go/generated"
	"kiezbox/internal/meshtastic"
	"kiezbox/logging"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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
	DBRetry(ctx context.Context, wg *sync.WaitGroup, db_client *db.InfluxDB)
	SetKiezboxValues(ctx context.Context, wg *sync.WaitGroup, control *generated.KiezboxMessage_Control)
	GetConfig(ctx context.Context, wg *sync.WaitGroup, interval time.Duration)
	ConfigWriter(ctx context.Context, wg *sync.WaitGroup)
	APIHandler(ctx context.Context, wg *sync.WaitGroup, r *gin.Engine)
}

// RunGoroutines orchestrates the goroutines that run the service.
func RunGoroutines(ctx context.Context, wg *sync.WaitGroup, device MeshtasticDevice, setTime bool, dbwriter bool, dbretry bool, db_client *db.InfluxDB) {
	// Launch goroutines
	wg.Add(1)
	go device.Writer(ctx, wg)
	wg.Add(1)
	go device.Heartbeat(ctx, wg, 30*time.Second)
	wg.Add(1)
	go device.Reader(ctx, wg)
	wg.Add(1)
	go device.MessageHandler(ctx, wg)
	wg.Add(1)
	go device.GetConfig(ctx, wg, 30*time.Second)
	wg.Add(1)
	go device.ConfigWriter(ctx, wg)

	if setTime {
		// We wait for the not info to set the time
		wg.Add(1)
		now := time.Now().Unix()
		control := meshtastic.BuildKiezboxControl(&now, nil) // Set time, mode is nil
		go device.SetKiezboxValues(ctx, wg, control)
	}

	// Process incoming KiexBox messages in its own goroutine
	if dbwriter {
		wg.Add(1)
		go device.DBWriter(ctx, wg, db_client)
		// } else {
		// 	mts.WaitInfo.Wait()
	}

	// Start the retry mechanism in its own goroutine
	if dbretry {
		wg.Add(1)
		go device.DBRetry(ctx, wg, db_client)
	}

	// Start the API in its own goroutine
	wg.Add(1)
	// Create a new Gin router
	r := gin.Default()
	// Register API routes
	routes.RegisterRoutes(r, device.(*meshtastic.MTSerial), ctx, wg)
	go device.APIHandler(ctx, wg, r)
}

func main() {
	logging.InitLogger(logging.LoggerConfig{
		Level:     slog.LevelInfo,
		Format:    "text",
		Filename:  "",
		AddSource: true,
	})

	slog.Info("Logger initialized", "app", "kiezbox-gateway-service")

	flag_settime := flag.Bool("settime", false, "Sets the RTC time to the system time at service startup")
	flag_dbwriter := flag.Bool("dbwriter", false, "Tells the service to run the dbwriter routine")
	flag_dbretry := flag.Bool("dbretry", false, "Tells the service to run the dbretry routine")
	flag_help := flag.Bool("help", false, "Prints the help info and exits")
	flag_serial_device := flag.String("dev", "/dev/ttyUSB0", "The serial device connecting us to the meshtastic device")
	flag_serial_baud := flag.Int("baud", 115200, "Baud rate of the serial device")
	flag_retry_time := flag.Int("retry", 10, "Time in seconds to retry writing to database")
	flag_cache_dir := flag.String("cache_dir", ".kb-dbcache", "Directory for caching points")
	flag_timeout := flag.Int("timeout", 3, "Database timeout in seconds")
	flag_api_port := flag.String("api_port", "9080", "API port")
	flag_api_sessiondir := flag.String("api_sessiondir", ".kb-session", "Path of the directory used for storing web client sessions")
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

	// TODO: portFactory is also defined in serial.go, this should be taken care of eventally
	portFactory := func(conf *serial.Config) (meshtastic.SerialPort, error) {
		return serial.OpenPort(conf)
	}
	mts.Init(*flag_serial_device, *flag_serial_baud, *flag_retry_time, *flag_api_port, portFactory, *flag_cache_dir, *flag_api_sessiondir)

	// Load InfluxDB configuration
	url, token, org, bucket := config.LoadConfig()

	// Initialize InfluxDB client
	db_client := db.CreateClient(url, token, org, bucket, *flag_timeout)
	defer db_client.Close()

	// Create a context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a WaitGroup to wait for the goroutines
	var wg sync.WaitGroup

	// Run the goroutines
	RunGoroutines(ctx, &wg, &mts, *flag_settime, *flag_dbwriter, *flag_dbretry, db_client)

	// Wait for all goroutines to finish
	wg.Wait()
}
