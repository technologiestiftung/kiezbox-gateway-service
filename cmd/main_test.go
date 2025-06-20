package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/mock"
	"github.com/tarm/serial"

	cfg "kiezbox/internal/config"
	"kiezbox/internal/db"
	"kiezbox/internal/github.com/meshtastic/go/generated"
	"kiezbox/internal/meshtastic"
)

// Mocks
type MockMTSerial struct {
	*meshtastic.MTSerial
	mock.Mock
}

func (m *MockMTSerial) Write(data []byte) (int, error) {
	args := m.Called(data)
	return args.Int(0), args.Error(1)
}

func (m *MockMTSerial) Close() error {
	return m.Called().Error(0)
}

func (m *MockMTSerial) Read(b []byte) (int, error) {
	args := m.Called(b)
	return args.Int(0), args.Error(1)
}

func (m *MockMTSerial) Writer(ctx context.Context, wg *sync.WaitGroup) {
	m.Called()
	wg.Done()
}

func (m *MockMTSerial) Heartbeat(ctx context.Context, wg *sync.WaitGroup, duration time.Duration) {
	m.Called(ctx, wg, duration)
	wg.Done()
}

func (m *MockMTSerial) Reader(ctx context.Context, wg *sync.WaitGroup) {
	m.Called(ctx, wg)
	wg.Done()
}

func (m *MockMTSerial) MessageHandler(ctx context.Context, wg *sync.WaitGroup) {
	m.Called(ctx, wg)
	wg.Done()
}

func (m *MockMTSerial) DBWriter(ctx context.Context, wg *sync.WaitGroup, db_client *db.InfluxDB) {
	m.Called(ctx, wg)
	wg.Done()
}

func (m *MockMTSerial) DBRetry(ctx context.Context, wg *sync.WaitGroup, db_client *db.InfluxDB) {
	m.Called(ctx, wg)
	wg.Done()
}

func (m *MockMTSerial) SetKiezboxValues(ctx context.Context, wg *sync.WaitGroup, control *generated.KiezboxMessage_Control) {
	m.Called(ctx, wg, control)
	wg.Done()
}

func (m *MockMTSerial) GetConfig(ctx context.Context, wg *sync.WaitGroup, duration time.Duration) {
	m.Called(ctx, wg, duration)
	wg.Done()
}

func (m *MockMTSerial) ConfigWriter(ctx context.Context, wg *sync.WaitGroup) {
	m.Called(ctx, wg)
	wg.Done()
}

func (m *MockMTSerial) APIHandler(ctx context.Context, wg *sync.WaitGroup, r *gin.Engine) {
	m.Called(ctx, wg)
	wg.Done()
}

func TestRunGoroutines(t *testing.T) {
	// Load default config values
	// We may (need to) overwrite some config for testing
	cfg.LoadConfigNoFail()
	// Setup
	mockMTSerial := &MockMTSerial{}
	mockMTSerial.On("Write", mock.Anything).Return(0, nil)
	mockMTSerial.On("Read", mock.Anything).Return(0, nil)
	mockMTSerial.On("Close").Return(nil)
	mockMTSerial.On("Writer", mock.Anything, mock.Anything).Return(nil)
	mockMTSerial.On("Heartbeat", mock.Anything, mock.Anything, time.Duration(30*time.Second)).Return(nil)
	mockMTSerial.On("Reader", mock.Anything, mock.Anything).Return(nil)
	mockMTSerial.On("MessageHandler", mock.Anything, mock.Anything).Return(nil)
	mockMTSerial.On("DBWriter", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockMTSerial.On("DBRetry", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockMTSerial.On("SetKiezboxControlValue", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockMTSerial.On("GetConfig", mock.Anything, mock.Anything, time.Duration(30*time.Second)).Return(nil)
	mockMTSerial.On("ConfigWriter", mock.Anything, mock.Anything).Return(nil)
	mockMTSerial.On("APIHandler", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	portFactory := func(conf *serial.Config) (meshtastic.SerialPort, error) {
		return mockMTSerial, nil
	}

	db_client := &db.InfluxDB{} // Mocked or a real one if needed

	// Initialize with a mock serial port
	var mts meshtastic.MTSerial
	mts.Init(portFactory)

	// Create a context with cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Create a WaitGroup to wait for the goroutines
	var wg sync.WaitGroup

	// Run the function under test
	RunGoroutines(ctx, &wg, mockMTSerial, db_client)

	// Cancel the context after a small interval
	time.Sleep(time.Millisecond * 1)
	cancel()

	// Assertions to check if the expected functions were called
	mockMTSerial.AssertCalled(t, "Writer")
	mockMTSerial.AssertCalled(t, "Heartbeat", mock.Anything, mock.Anything, time.Duration(30*time.Second))
	mockMTSerial.AssertCalled(t, "Reader", mock.Anything, mock.Anything)
	mockMTSerial.AssertCalled(t, "MessageHandler", mock.Anything, mock.Anything)
	mockMTSerial.AssertCalled(t, "DBWriter", mock.Anything, mock.Anything, mock.Anything)
	mockMTSerial.AssertCalled(t, "DBRetry", mock.Anything, mock.Anything, mock.Anything)
	mockMTSerial.AssertCalled(t, "SetKiezboxControlValue", mock.Anything, mock.Anything, mock.Anything)
	mockMTSerial.AssertCalled(t, "GetConfig", mock.Anything, mock.Anything, time.Duration(30*time.Second))
	mockMTSerial.AssertCalled(t, "ConfigWriter", mock.Anything, mock.Anything)
	mockMTSerial.AssertCalled(t, "APIHandler", mock.Anything, mock.Anything, mock.Anything)

	// Wait for all goroutines to finish
	wg.Wait()
}
