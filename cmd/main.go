package main

import (
	"context"
	"kiezbox/api/routes"

	cfg "kiezbox/internal/config"
	"kiezbox/internal/db"
	"kiezbox/internal/github.com/meshtastic/go/generated"
	"kiezbox/internal/meshtastic"
	"kiezbox/logging"
	"log/slog"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
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
	SetKiezboxControlValue(ctx context.Context, wg *sync.WaitGroup, control *generated.KiezboxMessage_Control)
	GetConfig(ctx context.Context, wg *sync.WaitGroup, interval time.Duration)
	ConfigWriter(ctx context.Context, wg *sync.WaitGroup)
	APIHandler(ctx context.Context, wg *sync.WaitGroup, r *gin.Engine)
}

// RunGoroutines orchestrates the goroutines that run the service.
func RunGoroutines(ctx context.Context, wg *sync.WaitGroup, device MeshtasticDevice, db_client *db.InfluxDB) {
	// Launch goroutines
	//TODO: refactor this function, as it is a little clumsy with all the manual waitgroup stuff
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

	if cfg.Cfg.SetTime {
		// We wait for the not info to set the time
		wg.Add(1)
		now := time.Now().Unix()
		message := &generated.KiezboxMessage_Control{
			Set: &generated.KiezboxMessage_Control_UnixTime{
				UnixTime: now,
			},
		}
		go device.SetKiezboxControlValue(ctx, wg, message)
	}

	// Process incoming KiezBox messages in its own goroutine
	if cfg.Cfg.DbWriter {
		wg.Add(1)
		go device.DBWriter(ctx, wg, db_client)
		// } else {
		// 	mts.WaitInfo.Wait()
	}

	// Start the retry mechanism in its own goroutine
	if cfg.Cfg.DbRetry {
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
	cfg.LoadConfig()
	logging.InitLogger(logging.LoggerConfig{
		Level:     slog.Level(cfg.Cfg.LogLevel),
		Format:    "text",
		LogFile:   cfg.Cfg.LogFile,
		LogToFile: cfg.Cfg.LogToFile,
		AddSource: cfg.Cfg.LogSource,
		ShortPath: cfg.Cfg.LogShortPath,
	})

	slog.Info("Logger initialized", "app", "kiezbox-gateway-service")
	slog.Debug("Service configuration", "cfg", cfg.Cfg)

	// Initialize meshtastic serial connection
	var mts meshtastic.MTSerial

	mts.Init(meshtastic.CreateSerialPort)

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
