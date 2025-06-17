# Instructions

Initialize the database.

```
docker compose up
```

## Protobuf > pb.go generation

First, update the protobufs submodule.

```
git submodule update --init
```

Then run this command tu generate the Go code based on the protobuf.

```
protoc -I=protobufs/ --go_out=internal protobufs/meshtastic/*.proto protobufs/nanopb.proto
```

## Build

Because we also build this for OpenWrt (23.05), we only have go in version 1.21.13 available.
To install and use go in the correct version:

```
go install golang.org/dl/go1.21.13@latest
~/go/bin/go1.21.13 download
```

Executing go should now always be done like this `~/go/bin/go1.21.13 version` instead of like this `go version`.
Executing `go mod tidy` e.g. may break the dependecies for the OpenWrt build if you system go is not in the right version.

Create a kiezbox module.

```
go mod init kiezbox
```

Get the serial library.
```
go get github.com/tarm/serial
```

Install dependencies.

```
go mod tidy
```

Run the main fail to test the module.
```
go run cmd/main.go
```

## Unittests

To run the tests.

```
go test ./...
```

To get a visual representation of code coverage in the tests.

```
go tool cover -html=coverage.out
```

## API

To run the API:

```
go run cmd/main.go
```

You can send API requests using cURL, like in the following examples:

```
curl -X POST "http://localhost:9080/admin/control?key=mode&value=emergency"
```

## Logging

This project uses Go's [log/slog](https://pkg.go.dev/log/slog) package for all logging.  
The logger is initialized at startup and set as the default, so you can use `slog.Info`, `slog.Error`, etc., anywhere in the codebase.

- **Always use structured logging:**  
  Use key-value pairs to provide context for each log message.
- **Log errors with the `"err"` key:**  
  Always include the error object with the key `"err"` for error logs.
- **Use clear, descriptive parameter names** for context (e.g., `"userID"`, `"file"`, `"id"`, `"request"`).
- **Do not use** `fmt.Println` or `log.Println` for application logs.

**Example log call:**
```go
slog.Error("Failed to open file", "file", filePath, "err", err)
```

**Example output (text format):**
```
time=2025-05-28T13:37:01.456Z level=ERROR msg="Failed to open file" file="/tmp/session1.json" err="open /tmp/session1.json: no such file or directory" source=api/handlers/asterisk.go:42
```

See `logging/logging.go` for configuration details.
