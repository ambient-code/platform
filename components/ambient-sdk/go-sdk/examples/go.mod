module github.com/ambient-code/platform/components/ambient-sdk/go-sdk/examples

go 1.24.0

toolchain go1.24.9

replace github.com/ambient-code/platform/components/ambient-sdk/go-sdk => ../

require github.com/ambient-code/platform/components/ambient-sdk/go-sdk v0.0.0-00010101000000-000000000000

require (
	github.com/ambient-code/platform/components/ambient-api-server v0.0.0-20260304211549-047314a7664b // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.32.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
	google.golang.org/grpc v1.79.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
