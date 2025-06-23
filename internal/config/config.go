package config

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/BoRuDar/configuration/v4"
	"github.com/joho/godotenv"
)

type GatewayConfig struct {
	SetTime       bool          `flag:"settime|true|Sets the RTC time of the device to the system time at service startup" default:"true"`
	DbWriter      bool          `flag:"dbwriter|true|Enables the dbwriter routine, which forwards sensor datapoints from meshtastic to the influxdb" default:"true"`
	DbRetry       bool          `flag:"dbretry|true|Enables the dbretry routine, which preiodically moves cached datapoints into the influxdb" default:"true"`
	DbUrl         string        `flag:"dburl||Full URL of the influxdb" env:"INFLUXDB_URL"`
	DbToken       string        `flag:"dbtoken||API token for the influxdb" env:"INFLUXDB_TOKEN"`
	DbOrg         string        `flag:"dborg||Organisation to use in the influxdb" env:"INFLUXDB_ORG"`
	DbBucket      string        `flag:"dbbucket||Bucket to use in the influxdb" env:"INFLUXDB_BUCKET"`
	SerialDevice  string        `flag:"serial_dev|/dev/ttyUSB0|The serial device connected to the meshtastic device" default:"/dev/ttyUSB0"`
	SerialBaud    int           `flag:"serial_baud|115200|Baud rate of the serial device" default:"115200"`
	RetryInterval time.Duration `flag:"retry_interval|60s|Time interval (as time.Duration) for the dbretry delay" default:"60s"`
	CacheDir      string        `flag:"cache_dir|.kb-dbcache|Directory for caching datapoints" default:".kb-dbcache"`
	DbTimeout     time.Duration `flag:"db_timeout|5s|Database timeout (as time.Duration)" default:"5s"`
	ApiPort       string        `flag:"api_port|9080|Port to use for the gateway service HTTP API" default:"9080"`
	SessionDir    string        `flag:"api_sessiondir|.kb-session|Directory for storing emergency call user sessions" default:".kb-session"`
	LogLevel      int           `flag:"log_level|0|Loglevel (int) as defined by go slog" default:"0"` //slog logging levels constants are defined here. 0 is LevelInfo > https://pkg.go.dev/log/slog#LevelInfo
	LogFile       string        `flag:"log_file|.kb-gwlog|Log file for slog" default:".kb-gwlog"`
	LogToFile     bool          `flag:"log_tofile|false|Enables logging to the logfile instead of standard output" default:"false"`
	LogSource     bool          `flag:"log_source|true|Enables logging the filename with slog" default:"true"`
	LogShortPath  bool          `flag:"log_shortpath|true|Enables short filename format (basename only) for slog" default:"true"`
	LogSerial     bool          `flag:"log_serial|false|Enables logging the serial debug of the meshtastic device" default:"false"`
	SipTrunkBase  string        `uci:"trunk_base|2|Sup trunk base phone number" env:"SIP_TRUNK_BASE" default:"2"`
	BoxLat        float32       `uci:"geo_lat|0|Geolocation latitude of the box" env:"BOX_LAT" default:"0"`
	BoxLon        float32       `uci:"geo_lon|0|Geolocation longitude of the box" env:"BOX_LON" default:"0"`
	Mode          int           `flag:"mode|0|Default device mode as defined in the protobuf" env:"KB_MODE" default:"0"`
	ModeOverride  bool          `flag:"mode_override|false|Enables overwriting the device mode with the cli/env flag" env:"KB_MODE_OVERRIDE" default:"false"`
	CorsLocalhost bool          `flag:"cors_localhost|false|Adds CORS header for local testing only" env:"CORS_LOCALHOST" default:"false"`
}

// Global gateway service config
var Cfg GatewayConfig

// Make sure that the config is only loaded once
var once sync.Once

// Wrapper to load configuration from different sources
// Fails if a config option could not be loaded from any source
func LoadConfig() {
	loadConfig(false)
}

// Similar wrapper, but does not fail if configuration options are missing
// mainly used in unit tests, where the environment may be different
// Warnings are printed for missing options
func LoadConfigNoFail() {
	loadConfig(true)
}

func loadConfig(nofail bool) {
	once.Do(func() {
		// Also load environment variables from .env file
		cwd, _ := os.Getwd()
		err := godotenv.Load()
		if err != nil {
			log.Printf("Failed loading %s.env file: %v", cwd, err)
		}
		//Configuration value priority:
		// 1. cli argurments
		// 2. uci values
		// 3. environment variables
		// 4. default values
		configurator := configuration.New(
			&Cfg,
			configuration.NewFlagProvider(),
			NewUciProvider("kb.main."),
			configuration.NewEnvProvider(),
			configuration.NewDefaultProvider(),
		)
		if nofail {
			configurator.SetOptions(
				configuration.OnFailFnOpt(func(err error) {
					log.Println(err)
				}),
			)
		}
		if err := configurator.InitValues(); err != nil {
			log.Fatal("Configuration error: ", err)
		}
	})
}
