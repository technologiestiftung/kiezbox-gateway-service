package db

import (
	"context"
	c "kiezbox/internal/config"
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
	client := influxdb.NewClient(c.Cfg.DbUrl, c.Cfg.DbToken)

	// Check if the client is working
	_, err := client.Health(context.Background())
	if err != nil {
		log.Printf("Error connecting to InfluxDB: %v", err)
	}

	// Initialize WriteAPI and QueryAPI once
	writeAPI := client.WriteAPIBlocking(c.Cfg.DbOrg, c.Cfg.DbBucket)
	queryAPI := client.QueryAPI(c.Cfg.DbOrg)

	return &InfluxDB{
		Client:   client,
		WriteAPI: writeAPI,
		QueryAPI: queryAPI,
		Org:      c.Cfg.DbOrg,
		Bucket:   c.Cfg.DbBucket,
		Timeout:  c.Cfg.DbTimeout,
	}
}

// Close the InfluxDB client when no longer needed
func (db *InfluxDB) Close() {
	db.Client.Close()
}
