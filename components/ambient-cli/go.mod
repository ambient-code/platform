module github.com/ambient-code/platform/components/ambient-cli

go 1.24.4

require (
	github.com/ambient-code/platform/components/ambient-sdk/go-sdk v0.0.0
	github.com/golang-jwt/jwt/v4 v4.5.2
	github.com/spf13/cobra v1.9.1
	golang.org/x/term v0.28.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	golang.org/x/sys v0.29.0 // indirect
)

replace github.com/ambient-code/platform/components/ambient-sdk/go-sdk => ../ambient-sdk/go-sdk
