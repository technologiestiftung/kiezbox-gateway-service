# Instructions

Initialize the database.

```
docker compose up
```

Update the protobufs submodule, if you want to generate the 

```
git submodule update --init
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
