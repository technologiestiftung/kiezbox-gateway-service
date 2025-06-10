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
	SetTime       bool          `flag:"settime" default:"true"`
	DbWriter      bool          `flag:"dbwriter" default:"true"`
	DbRetry       bool          `flag:"dbretry" default:"true"`
	DbUrl         string        `flag:"dburl" env:"INFLUXDB_URL"`
	DbToken       string        `flag:"dbtoken" env:"INFLUXDB_TOKEN"`
	DbOrg         string        `flag:"dborg" env:"INFLUXDB_ORG"`
	DbBucket      string        `flag:"dbbucket" env:"INFLUXDB_BUCKET"`
	SerialDevice  string        `flag:"serial_dev" default:"/dev/ttyUSB0"`
	SerialBaud    int           `flag:"serial_baud" default:"115200"`
	RetryInterval time.Duration `flag:"retry_interval" default:"60s"`
	CacheDir      string        `flag:"cache_dir" default:".kb-dbcache"`
	DbTimeout     time.Duration `flag:"db_timeout" default:"5s"`
	ApiPort       string        `flag:"api_port" default:"9080"`
	SessionDir    string        `flag:"api_sessiondir" default:".kb-session"`
	LogLevel      int           `flag:"log_level" default:"0"` //slog logging levels constants are defined here. 0 is LevelInfo > https://pkg.go.dev/log/slog#LevelInfo
	LogFile       string        `flag:"log_file" default:".kb-gwlog"`
	LogToFile     bool          `flag:"log_tofile" default:"false"`
	LogSource     bool          `flag:"log_source" default:"true"`
	LogShortPath  bool          `flag:"log_shortpath" default:"true"`
	SipTrunkBase  string        `uci:"trunk_base" env:"SIP_TRUNK_BASE" default:"2"`
	BoxLat        float32       `uci:"geo_lat" env:"BOX_LAT" default:"0"`
	BoxLon        float32       `uci:"geo_lon" env:"BOX_LON" default:"0"`
	Mode          int           `uci:"mode" env:"KB_MODE" default:"0"`
	ModeOverride  bool          `uci:"mode_override" env:"KB_MODE_OVERRIDE" default:"false"`
	CorsLocalhost bool          `uci:"cors_localhost" env:"CORS_LOCALHOST" default:"false"`
}

// Global gateway service config
var Cfg GatewayConfig

// Make sure that the config is only loaded once
var once sync.Once

func LoadConfig() {
	loadConfig(false)
}

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
