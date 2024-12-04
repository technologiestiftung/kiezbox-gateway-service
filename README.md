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
protoc -I=. --go_out=internal protobufs/meshtastic/kiezbox_control.proto
```

Create a kiezbox module.

```
go mod init kiezbox
```

Install dependencies.

```
go mod tidy
```

Run the main fail to test the module.
```
go run cmd/main.go
```

## Run unittests

```
go test ./...
```
