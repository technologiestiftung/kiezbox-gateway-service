package db

import (
	"context"
	"log"
	"time"

	influxdb "github.com/influxdata/influxdb-client-go/v2"
	influxdb_api "github.com/influxdata/influxdb-client-go/v2/api"
)

const Timeout = 1 * time.Second

type InfluxDB struct {
	Client   influxdb.Client
	WriteAPI influxdb_api.WriteAPIBlocking
	QueryAPI influxdb_api.QueryAPI
	Org      string
	Bucket   string
	Timeout  time.Duration
}

// CreateClient initializes the InfluxDB client and APIs
func CreateClient(url, token, org, bucket string) *InfluxDB {
	// Create a new InfluxDB client
	client := influxdb.NewClient(url, token)

	// Check if the client is working
	_, err := client.Health(context.Background())
	if err != nil {
		log.Fatalf("Error connecting to InfluxDB: %v", err)
	}

	// Initialize WriteAPI and QueryAPI once
	writeAPI := client.WriteAPIBlocking(org, bucket)
	queryAPI := client.QueryAPI(org)

	return &InfluxDB{
		Client:   client,
		WriteAPI: writeAPI,
		QueryAPI: queryAPI,
		Org:      org,
		Bucket:   bucket,
		Timeout: Timeout,
	}
}

// Close the InfluxDB client when no longer needed
func (db *InfluxDB) Close() {
	db.Client.Close()
}
