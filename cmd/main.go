package main

import (
	"context"
	"kiezbox/internal/config"
	"kiezbox/internal/db"
	"kiezbox/internal/meshtastic"
	"kiezbox/logging"
	"log"
	"log/slog"
	"sync"
	"time"

	"github.com/BoRuDar/configuration/v4"
	"github.com/tarm/serial"
)

type Config struct {
    SetTime bool `flag:"settime" default:"true"`
    DbWriter bool `flag:"dbwriter" default:"true"`
    DbRetry bool `flag:"dbretry" default:"true"`
    SerialDevice string `flag:"serial_dev" default:"/dev/ttyUSB0"`
    SerialBaud int `flag:"serial_baud" default:"115200"`
    RetryInterval int `flag:"retry_interval" default:"60"`
    CacheDir string `flag:"cache_dir" default:".kb-dbcache"`
    DbTimeout int `flag:"db_timeout" default:"5"`
    ApiPort string `flag:"api_port" default:"9080"`
    SessionDir string `flag:"api_sessiondir" default:".kb-session"`
}

// Global gateway service config
var gwConfig Config

// Using an interface as an intermediate layer instead of calling the meshtastic functions directly
// allows for dependency injection, essential for unittesting.
type MeshtasticDevice interface {
	Writer(ctx context.Context, wg *sync.WaitGroup)
	Heartbeat(ctx context.Context, wg *sync.WaitGroup, interval time.Duration)
	Reader(ctx context.Context, wg *sync.WaitGroup)
	MessageHandler(ctx context.Context, wg *sync.WaitGroup)
	DBWriter(ctx context.Context, wg *sync.WaitGroup, db_client *db.InfluxDB)
	DBRetry(ctx context.Context, wg *sync.WaitGroup, db_client *db.InfluxDB)
	Settime(ctx context.Context, wg *sync.WaitGroup, time int64)
	GetConfig(ctx context.Context, wg *sync.WaitGroup, interval time.Duration)
	ConfigWriter(ctx context.Context, wg *sync.WaitGroup)
	APIHandler(ctx context.Context, wg *sync.WaitGroup)
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
		go device.Settime(ctx, wg, time.Now().Unix())
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
	go device.APIHandler(ctx, wg)
}

func main() {
	//Configuration value priority:
	// 1. cli argurments
	// 2. environment variables
	// 3. default values
	configurator := configuration.New(
		&gwConfig,
		configuration.NewFlagProvider(),
		configuration.NewEnvProvider(),
		configuration.NewDefaultProvider(),
	)
	if err := configurator.InitValues(); err != nil {
		log.Fatal("Configuration error: ", err)
	}
	logging.InitLogger(logging.LoggerConfig{
		Level:     slog.LevelInfo,
		Format:    "text",
		Filename:  "",
		AddSource: true,
		ShortPath: true,
	})

	slog.Info("Logger initialized", "app", "kiezbox-gateway-service")
	slog.Debug("Service configuration", "cfg", gwConfig)

	// Initialize meshtastic serial connection
	var mts meshtastic.MTSerial

	// TODO: portFactory is also defined in serial.go, this should be taken care of eventally
	portFactory := func(conf *serial.Config) (meshtastic.SerialPort, error) {
		return serial.OpenPort(conf)
	}
	mts.Init(gwConfig.SerialDevice, gwConfig.SerialBaud, gwConfig.RetryInterval, gwConfig.ApiPort, portFactory, gwConfig.CacheDir, gwConfig.SessionDir)

	// Load InfluxDB configuration
	url, token, org, bucket := config.LoadConfig()

	// Initialize InfluxDB client
	db_client := db.CreateClient(url, token, org, bucket, gwConfig.DbTimeout)
	defer db_client.Close()

	// Create a context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a WaitGroup to wait for the goroutines
	var wg sync.WaitGroup

	// Run the goroutines
	RunGoroutines(ctx, &wg, &mts, gwConfig.SetTime, gwConfig.DbWriter, gwConfig.DbRetry, db_client)

	// Wait for all goroutines to finish
	wg.Wait()
}
