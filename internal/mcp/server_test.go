package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// testServer creates a server with mock I/O for testing
func testServer(input string) (*Server, *bytes.Buffer) {
	output := &bytes.Buffer{}
	s := &Server{
		reader: bufio.NewReader(strings.NewReader(input)),
		writer: output,
		tools:  make(map[string]ToolHandler),
	}
	s.registerTools()
	return s, output
}

func TestHandleInitialize(t *testing.T) {
	request := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}` + "\n"
	s, output := testServer(request)

	s.Run()

	var resp JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", resp.Result)
	}

	if result["protocolVersion"] != ProtocolVersion {
		t.Errorf("Expected protocol version %s, got %v", ProtocolVersion, result["protocolVersion"])
	}

	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected serverInfo map")
	}

	if serverInfo["name"] != ServerName {
		t.Errorf("Expected server name %s, got %v", ServerName, serverInfo["name"])
	}
}

func TestHandleListTools(t *testing.T) {
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}` + "\n"
	s, output := testServer(request)

	s.Run()

	var resp JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", resp.Result)
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("Expected tools array")
	}

	// Check that we have expected tools
	expectedTools := []string{
		"checkpoint_create",
		"checkpoint_list",
		"checkpoint_rollback",
		"checkpoint_status",
		"checkpoint_delete",
		"checkpoint_diff",
		"checkpoint_tag",
		"checkpoint_search",
		"checkpoint_compress",
		"checkpoint_decompress",
	}

	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolMap := tool.(map[string]interface{})
		toolNames[toolMap["name"].(string)] = true
	}

	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Expected tool %s not found", expected)
		}
	}
}

func TestHandlePing(t *testing.T) {
	request := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}` + "\n"
	s, output := testServer(request)

	s.Run()

	var resp JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("Unexpected error: %v", resp.Error)
	}
}

func TestHandleUnknownMethod(t *testing.T) {
	request := `{"jsonrpc":"2.0","id":1,"method":"unknown_method","params":{}}` + "\n"
	s, output := testServer(request)

	s.Run()

	var resp JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error for unknown method")
	}

	if resp.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", resp.Error.Code)
	}
}

func TestHandleInvalidJSON(t *testing.T) {
	request := `{invalid json}` + "\n"
	s, output := testServer(request)

	s.Run()

	var resp JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Error("Expected error for invalid JSON")
	}

	if resp.Error.Code != -32700 {
		t.Errorf("Expected parse error code -32700, got %d", resp.Error.Code)
	}
}

func TestHandleCallToolUnknown(t *testing.T) {
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"unknown_tool","arguments":{}}}` + "\n"
	s, output := testServer(request)

	s.Run()

	var resp JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	// Tool errors are returned as successful responses with IsError flag
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result")
	}

	isError, ok := result["isError"].(bool)
	if !ok || !isError {
		t.Error("Expected isError to be true for unknown tool")
	}
}

func TestToolsRegistered(t *testing.T) {
	s, _ := testServer("")

	expectedTools := []string{
		"checkpoint_create",
		"checkpoint_list",
		"checkpoint_rollback",
		"checkpoint_status",
		"checkpoint_delete",
		"checkpoint_diff",
		"checkpoint_tag",
		"checkpoint_search",
		"checkpoint_compress",
		"checkpoint_decompress",
	}

	for _, toolName := range expectedTools {
		if _, ok := s.tools[toolName]; !ok {
			t.Errorf("Tool %s not registered", toolName)
		}
	}
}

func TestMultipleRequests(t *testing.T) {
	requests := `{"jsonrpc":"2.0","id":1,"method":"ping","params":{}}
{"jsonrpc":"2.0","id":2,"method":"ping","params":{}}
{"jsonrpc":"2.0","id":3,"method":"ping","params":{}}
`
	s, output := testServer(requests)

	s.Run()

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 3 {
		t.Errorf("Expected 3 responses, got %d", len(lines))
	}

	for i, line := range lines {
		var resp JSONRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Errorf("Failed to parse response %d: %v", i+1, err)
		}
		if resp.Error != nil {
			t.Errorf("Unexpected error in response %d: %v", i+1, resp.Error)
		}
	}
}

// Benchmark tests
func BenchmarkHandleInitialize(b *testing.B) {
	request := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}` + "\n"

	for i := 0; i < b.N; i++ {
		s, _ := testServer(request)
		s.Run()
	}
}

func BenchmarkHandleListTools(b *testing.B) {
	request := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}` + "\n"

	for i := 0; i < b.N; i++ {
		s, _ := testServer(request)
		s.Run()
	}
}
