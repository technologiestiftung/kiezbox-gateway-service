package main

import (
	"context"
	c "kiezbox/internal/config"
	"kiezbox/internal/db"
	"kiezbox/internal/meshtastic"
	"kiezbox/logging"
	"log/slog"
	"sync"
	"time"
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
	Settime(ctx context.Context, wg *sync.WaitGroup, time int64)
	GetConfig(ctx context.Context, wg *sync.WaitGroup, interval time.Duration)
	ConfigWriter(ctx context.Context, wg *sync.WaitGroup)
	APIHandler(ctx context.Context, wg *sync.WaitGroup)
}

// RunGoroutines orchestrates the goroutines that run the service.
func RunGoroutines(ctx context.Context, wg *sync.WaitGroup, device MeshtasticDevice, db_client *db.InfluxDB) {
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

	if c.Cfg.SetTime {
		// We wait for the not info to set the time
		wg.Add(1)
		go device.Settime(ctx, wg, time.Now().Unix())
	}

	// Process incoming KiexBox messages in its own goroutine
	if c.Cfg.DbWriter {
		wg.Add(1)
		go device.DBWriter(ctx, wg, db_client)
		// } else {
		// 	mts.WaitInfo.Wait()
	}

	// Start the retry mechanism in its own goroutine
	if c.Cfg.DbRetry {
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
	c.LoadConfig()
	logging.InitLogger(logging.LoggerConfig{
		Level:     slog.Level(c.Cfg.LogLevel),
		Format:    "text",
		LogFile:  c.Cfg.LogFile,
		LogToFile:  c.Cfg.LogToFile,
		AddSource: c.Cfg.LogSource,
		ShortPath: c.Cfg.LogShortPath,
	})

	slog.Info("Logger initialized", "app", "kiezbox-gateway-service")
	slog.Debug("Service configuration", "cfg", c.Cfg)

	// Initialize meshtastic serial connection
	var mts meshtastic.MTSerial

	mts.Init(c.Cfg.SerialDevice, c.Cfg.SerialBaud, c.Cfg.RetryInterval, c.Cfg.ApiPort, meshtastic.CreateSerialPort, c.Cfg.CacheDir, c.Cfg.SessionDir)

	// Initialize InfluxDB client
	db_client := db.CreateClient()
	defer db_client.Close()

	// Create a context with cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a WaitGroup to wait for the goroutines
	var wg sync.WaitGroup

	// Run the goroutines
	RunGoroutines(ctx, &wg, &mts, db_client)

	// Wait for all goroutines to finish
	wg.Wait()
}
