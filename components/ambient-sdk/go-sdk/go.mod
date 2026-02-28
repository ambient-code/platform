module github.com/ambient-code/platform/components/ambient-sdk/go-sdk

go 1.24.0

toolchain go1.24.9

require (
	github.com/ambient/platform/components/ambient-api-server v0.0.0
	google.golang.org/grpc v1.75.1
)

require (
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251014184007-4626949a642f // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/ambient/platform/components/ambient-api-server => ../../ambient-api-server
