package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx := context.Background()

	mcpA := getEnv(
		"MCP_A_URL",
		"http://service-a:8080/mcp",
	)

	mcpB := getEnv(
		"MCP_B_URL",
		"http://service-b:8080/mcp",
	)

	log.Println("=== AGENT START ===")

	log.Printf("MCP A: %s", mcpA)
	log.Printf("MCP B: %s", mcpB)

	if err := callTool(
		ctx,
		"agent-to-a",
		mcpA,
		"ask_service_b",
	); err != nil {
		log.Fatal(err)
	}

	if err := callTool(
		ctx,
		"agent-to-b",
		mcpB,
		"ask_service_a",
	); err != nil {
		log.Fatal(err)
	}

	log.Println("=== AGENT FINISHED ===")
}

func callTool(
	ctx context.Context,
	clientName string,
	endpoint string,
	toolName string,
) error {

	client := mcp.NewClient(
		&mcp.Implementation{
			Name:    clientName,
			Version: "v0.1.0",
		},
		nil,
	)

	transport := &mcp.StreamableClientTransport{
		Endpoint: endpoint,
	}

	session, err := client.Connect(
		ctx,
		transport,
		nil,
	)

	if err != nil {
		return fmt.Errorf(
			"erro conectando em %s: %w",
			endpoint,
			err,
		)
	}

	defer session.Close()

	log.Printf(
		"chamando MCP tool %s em %s",
		toolName,
		endpoint,
	)

	result, err := session.CallTool(
		ctx,
		&mcp.CallToolParams{
			Name:      toolName,
			Arguments: map[string]any{},
		},
	)

	if err != nil {
		return err
	}

	if result.IsError {
		return fmt.Errorf(
			"tool %s retornou erro",
			toolName,
		)
	}

	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			log.Printf(
				"resultado: %s",
				text.Text,
			)
		}
	}

	return nil
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)

	if value == "" {
		return fallback
	}

	return value
}
