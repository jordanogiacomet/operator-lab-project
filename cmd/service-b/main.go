package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type EmptyInput struct{}

type ToolOutput struct {
	Result string `json:"result" jsonschema:"resultado da operação"`
}

func main() {
	serviceAURL := getEnv(
		"SERVICE_A_URL",
		"http://localhost:8080",
	)

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "service-b ok")
	})

	mux.HandleFunc("/ask-a", func(w http.ResponseWriter, r *http.Request) {
		result, err := fetch(
			r.Context(),
			client,
			serviceAURL+"/health",
		)

		if err != nil {
			http.Error(
				w,
				"erro ao chamar service-a: "+err.Error(),
				http.StatusBadGateway,
			)
			return
		}

		fmt.Fprintf(
			w,
			"service-b recebeu do service-a: %s\n",
			result,
		)
	})

	// MCP Server
	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    "service-b",
			Version: "v0.2.0",
		},
		nil,
	)

	mcp.AddTool(
		mcpServer,
		&mcp.Tool{
			Name:        "ask_service_a",
			Description: "Pergunta ao Service A se ele está funcionando",
		},
		func(
			ctx context.Context,
			req *mcp.CallToolRequest,
			input EmptyInput,
		) (*mcp.CallToolResult, ToolOutput, error) {

			result, err := fetch(
				ctx,
				client,
				serviceAURL+"/health",
			)

			if err != nil {
				return nil, ToolOutput{}, err
			}

			return nil, ToolOutput{
				Result: "Service A respondeu: " + result,
			}, nil
		},
	)

	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			return mcpServer
		},
		&mcp.StreamableHTTPOptions{
			Stateless: true,
		},
	)

	mux.Handle("/mcp", mcpHandler)

	log.Printf(
		"service-b rodando na porta 8082 | service-a=%s",
		serviceAURL,
	)

	log.Printf("MCP disponível em /mcp")

	if err := http.ListenAndServe(":8082", mux); err != nil {
		log.Fatal(err)
	}
}

func fetch(
	ctx context.Context,
	client *http.Client,
	url string,
) (string, error) {

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)

	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf(
			"status inesperado: %s",
			resp.Status,
		)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
