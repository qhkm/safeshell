package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

const (
	ProtocolVersion = "2024-11-05"
	ServerName      = "safeshell"
	ServerVersion   = "0.1.0"
)

type Server struct {
	reader  *bufio.Reader
	writer  io.Writer
	mu      sync.Mutex
	tools   map[string]ToolHandler
}

type ToolHandler func(args map[string]interface{}) (string, error)

func NewServer() *Server {
	s := &Server{
		reader: bufio.NewReader(os.Stdin),
		writer: os.Stdout,
		tools:  make(map[string]ToolHandler),
	}
	s.registerTools()
	return s
}

func (s *Server) Run() error {
	for {
		line, err := s.reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read error: %w", err)
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		s.handleRequest(&req)
	}
}

func (s *Server) handleRequest(req *JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "notifications/initialized":
		// No response needed for notifications
	case "tools/list":
		s.handleListTools(req)
	case "tools/call":
		s.handleCallTool(req)
	case "ping":
		s.sendResult(req.ID, map[string]interface{}{})
	default:
		s.sendError(req.ID, -32601, "Method not found", req.Method)
	}
}

func (s *Server) handleInitialize(req *JSONRPCRequest) {
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		ServerInfo: ServerInfo{
			Name:    ServerName,
			Version: ServerVersion,
		},
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{},
		},
	}
	s.sendResult(req.ID, result)
}

func (s *Server) handleListTools(req *JSONRPCRequest) {
	tools := []Tool{
		{
			Name:        "checkpoint_create",
			Description: "Create a checkpoint (backup) for specified files before performing a destructive operation. Use this BEFORE running rm, mv, or other dangerous commands.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"paths": {
						Type:        "array",
						Description: "List of file or directory paths to backup",
						Items:       &Items{Type: "string"},
					},
					"reason": {
						Type:        "string",
						Description: "Reason for creating checkpoint (e.g., 'before deleting build folder')",
					},
				},
				Required: []string{"paths"},
			},
		},
		{
			Name:        "checkpoint_list",
			Description: "List all available checkpoints. Shows checkpoint IDs, timestamps, commands, and file counts.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"limit": {
						Type:        "string",
						Description: "Maximum number of checkpoints to return (default: 10)",
					},
				},
			},
		},
		{
			Name:        "checkpoint_rollback",
			Description: "Rollback to a previous checkpoint, restoring all backed up files to their original locations.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Checkpoint ID to rollback to. Use 'latest' for most recent checkpoint.",
					},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "checkpoint_status",
			Description: "Get SafeShell status including total checkpoints, storage used, and configuration.",
			InputSchema: InputSchema{
				Type:       "object",
				Properties: map[string]Property{},
			},
		},
		{
			Name:        "checkpoint_delete",
			Description: "Delete a specific checkpoint by ID.",
			InputSchema: InputSchema{
				Type: "object",
				Properties: map[string]Property{
					"id": {
						Type:        "string",
						Description: "Checkpoint ID to delete",
					},
				},
				Required: []string{"id"},
			},
		},
	}

	s.sendResult(req.ID, ListToolsResult{Tools: tools})
}

func (s *Server) handleCallTool(req *JSONRPCRequest) {
	paramsBytes, err := json.Marshal(req.Params)
	if err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	var params CallToolParams
	if err := json.Unmarshal(paramsBytes, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	handler, ok := s.tools[params.Name]
	if !ok {
		s.sendToolError(req.ID, fmt.Sprintf("Unknown tool: %s", params.Name))
		return
	}

	result, err := handler(params.Arguments)
	if err != nil {
		s.sendToolError(req.ID, err.Error())
		return
	}

	s.sendResult(req.ID, CallToolResult{
		Content: []ContentBlock{
			{Type: "text", Text: result},
		},
	})
}

func (s *Server) sendResult(id interface{}, result interface{}) {
	s.send(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (s *Server) sendError(id interface{}, code int, message string, data interface{}) {
	s.send(JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	})
}

func (s *Server) sendToolError(id interface{}, message string) {
	s.sendResult(id, CallToolResult{
		Content: []ContentBlock{
			{Type: "text", Text: message},
		},
		IsError: true,
	})
}

func (s *Server) send(response JSONRPCResponse) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(response)
	if err != nil {
		return
	}

	fmt.Fprintf(s.writer, "%s\n", data)
}
