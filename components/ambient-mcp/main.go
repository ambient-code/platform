package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/ambient-code/platform/components/ambient-mcp/client"
)

func main() {
	apiURL := os.Getenv("AMBIENT_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	token := os.Getenv("AMBIENT_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "AMBIENT_TOKEN is required")
		os.Exit(1)
	}

	transport := os.Getenv("MCP_TRANSPORT")
	if transport == "" {
		transport = "stdio"
	}

	c := client.New(apiURL, token)
	s := newServer(c, transport)

	switch transport {
	case "stdio":
		if err := server.ServeStdio(s); err != nil {
			fmt.Fprintf(os.Stderr, "stdio server error: %v\n", err)
			os.Exit(1)
		}

	case "sse":
		bindAddr := os.Getenv("MCP_BIND_ADDR")
		if bindAddr == "" {
			bindAddr = ":8090"
		}
		sseServer := server.NewSSEServer(s,
			server.WithBaseURL("http://"+bindAddr),
			server.WithSSEEndpoint("/sse"),
			server.WithMessageEndpoint("/message"),
		)
		fmt.Fprintf(os.Stderr, "MCP server (SSE) listening on %s\n", bindAddr)
		if err := http.ListenAndServe(bindAddr, sseServer); err != nil {
			fmt.Fprintf(os.Stderr, "SSE server error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown MCP_TRANSPORT: %q (must be stdio or sse)\n", transport)
		os.Exit(1)
	}
}
