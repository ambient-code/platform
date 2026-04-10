package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/mark3labs/mcp-go/server"

	"github.com/ambient-code/platform/components/ambient-mcp/client"
	"github.com/ambient-code/platform/components/ambient-mcp/tokenexchange"
)

func main() {
	apiURL := os.Getenv("AMBIENT_API_URL")
	if apiURL == "" {
		apiURL = "http://localhost:8080"
	}

	transport := os.Getenv("MCP_TRANSPORT")
	if transport == "" {
		transport = "stdio"
	}

	cpTokenURL := os.Getenv("AMBIENT_CP_TOKEN_URL")
	cpPublicKey := os.Getenv("AMBIENT_CP_TOKEN_PUBLIC_KEY")
	sessionID := os.Getenv("SESSION_ID")

	var token string
	var exchanger *tokenexchange.Exchanger

	if cpTokenURL != "" && cpPublicKey != "" && sessionID != "" {
		var err error
		exchanger, err = tokenexchange.New(cpTokenURL, cpPublicKey, sessionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "token exchange init failed: %v\n", err)
			os.Exit(1)
		}
		token, err = exchanger.FetchToken()
		if err != nil {
			fmt.Fprintf(os.Stderr, "initial token fetch failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "bootstrapped token via CP token exchange")
	} else {
		token = os.Getenv("AMBIENT_TOKEN")
		if token == "" {
			fmt.Fprintln(os.Stderr, "AMBIENT_TOKEN is required when CP token exchange env vars are not set")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "using static AMBIENT_TOKEN (no CP token exchange)")
	}

	c := client.New(apiURL, token)

	if exchanger != nil {
		exchanger.OnRefresh(func(freshToken string) {
			c.SetToken(freshToken)
		})
		exchanger.StartBackgroundRefresh()
		defer exchanger.Stop()
	}

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
