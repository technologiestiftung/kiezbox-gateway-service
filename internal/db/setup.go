package db

import (
	"context"
	cfg "kiezbox/internal/config"
	"log"
	"time"

	influxdb "github.com/influxdata/influxdb-client-go/v2"
	influxdb_api "github.com/influxdata/influxdb-client-go/v2/api"
)

type InfluxDB struct {
	Client   influxdb.Client
	WriteAPI influxdb_api.WriteAPIBlocking
	QueryAPI influxdb_api.QueryAPI
	Org      string
	Bucket   string
	Timeout  time.Duration
}

// CreateClient initializes the InfluxDB client and APIs
func CreateClient() *InfluxDB {
	// Create a new InfluxDB client
	client := influxdb.NewClient(cfg.Cfg.DbUrl, cfg.Cfg.DbToken)

	// Check if the client is working
	_, err := client.Health(context.Background())
	if err != nil {
		log.Printf("Error connecting to InfluxDB: %v", err)
	}

	// Initialize WriteAPI and QueryAPI once
	writeAPI := client.WriteAPIBlocking(cfg.Cfg.DbOrg, cfg.Cfg.DbBucket)
	queryAPI := client.QueryAPI(cfg.Cfg.DbOrg)

	return &InfluxDB{
		Client:   client,
		WriteAPI: writeAPI,
		QueryAPI: queryAPI,
		Org:      cfg.Cfg.DbOrg,
		Bucket:   cfg.Cfg.DbBucket,
		Timeout:  cfg.Cfg.DbTimeout,
	}
}

// Close the InfluxDB client when no longer needed
func (db *InfluxDB) Close() {
	db.Client.Close()
}
