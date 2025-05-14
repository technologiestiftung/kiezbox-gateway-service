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

To update the API Swagger documentation (e.g., after adding new routes, modifying handlers or chaning Swagger metadata), run the following command from the root. This is the same as running `swag init` from the `api` directory:

```
swag init --output ./api/docs --dir ./cmd
```

To access the Swagger documentation, go to:

```
http://localhost:9080/swagger/index.html
```

To run the API:

```
go run cmd/main.go
```
