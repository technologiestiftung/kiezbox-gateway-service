package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/tarm/serial"

	"kiezbox/internal/db"
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

func (m *MockMTSerial) DBWriterRetry(ctx context.Context, wg *sync.WaitGroup, db_client *db.InfluxDB) {
	m.Called(ctx, wg)
	wg.Done()
}

func (m *MockMTSerial) Settime(ctx context.Context, wg *sync.WaitGroup, time int64) {
	m.Called(ctx, wg, time)
	wg.Done()
}

func (m *MockMTSerial) GetConfig(ctx context.Context, wg *sync.WaitGroup, duration time.Duration) {
	m.Called(ctx, wg, duration)
	wg.Done()
}

func TestRunGoroutines(t *testing.T) {
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
	mockMTSerial.On("DBWriterRetry", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockMTSerial.On("Settime", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockMTSerial.On("GetConfig", mock.Anything, mock.Anything, time.Duration(30*time.Second)).Return(nil)

	portFactory := func(conf *serial.Config) (meshtastic.SerialPort, error) {
		return mockMTSerial, nil
	}

	flag_settime := true
	flag_daemon := true
	db_client := &db.InfluxDB{} // Mocked or a real one if needed

	// Initialize with a mock serial port
	var mts meshtastic.MTSerial
	mts.Init("/dev/mockTTYUSB0", 115200, 10, portFactory)

	// Create a context with cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Create a WaitGroup to wait for the goroutines
	var wg sync.WaitGroup

	// Run the function under test
	RunGoroutines(ctx, &wg, mockMTSerial, flag_settime, flag_daemon, db_client)

	// Cancel the context after a small interval
	time.Sleep(time.Millisecond * 1)
	cancel()

	// Assertions to check if the expected functions were called
	mockMTSerial.AssertCalled(t, "Writer")
	mockMTSerial.AssertCalled(t, "Heartbeat", mock.Anything, mock.Anything, time.Duration(30*time.Second))
	mockMTSerial.AssertCalled(t, "Reader", mock.Anything, mock.Anything)
	mockMTSerial.AssertCalled(t, "MessageHandler", mock.Anything, mock.Anything)
	mockMTSerial.AssertCalled(t, "DBWriter", mock.Anything, mock.Anything, mock.Anything)
	mockMTSerial.AssertCalled(t, "DBWriterRetry", mock.Anything, mock.Anything, mock.Anything)
	mockMTSerial.AssertCalled(t, "Settime", mock.Anything, mock.Anything, mock.Anything)
	mockMTSerial.AssertCalled(t, "GetConfig", mock.Anything, mock.Anything, time.Duration(30*time.Second))

	// Wait for all goroutines to finish
	wg.Wait()
}
