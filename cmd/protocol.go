package cmd

import (
	"encoding/json"
	"fmt"
)

// ProtocolVersion is the canonical Naagmani plugin protocol version.
const ProtocolVersion = "naagmani.plugin/v1"

// Frozen Protocol v1 Method Names.
const (
	MethodRegister = "plugin.register"
	MethodHealth   = "plugin.health"
	MethodShutdown = "plugin.shutdown"
)

// RPCRequest is a JSON-RPC 2.0 request payload.
type RPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      int64           `json:"id"`
}

// RPCResponse is a JSON-RPC 2.0 response payload.
type RPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      int64           `json:"id"`
}

// RPCError represents a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("rpc error: code=%d message=%s", e.Code, e.Message)
}

// NewRPCRequest creates a new JSON-RPC 2.0 request payload.
func NewRPCRequest(id int64, method string, params any) (*RPCRequest, error) {
	rawParams, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshaling params: %w", err)
	}
	return &RPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  rawParams,
		ID:      id,
	}, nil
}

// RegisterParams is passed during "plugin.register" handshake.
type RegisterParams struct {
	PluginName      string            `json:"plugin_name"`
	PluginVersion   string            `json:"plugin_version"`
	APIVersion      string            `json:"api_version"`
	ProtocolVersion string            `json:"protocol_version,omitempty"`
	Config          json.RawMessage   `json:"config,omitempty"`
	Permissions     []string          `json:"permissions"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// HealthParams is passed during "plugin.health".
type HealthParams struct{}

// ShutdownParams is passed during "plugin.shutdown".
type ShutdownParams struct{}
