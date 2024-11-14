# Instructions

Automatic Go code generation based on the proto file.

```
protoc -I=. --go_out=. --go_opt=paths=source_relative status/status.proto
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
go run main.go
```
