package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	rpcParseError     = -32700
	rpcInvalidRequest = -32600
	rpcMethodNotFound = -32601
	rpcInvalidParams  = -32602
	rpcInternalError  = -32603
)

func (s *Server) ServeStdio(ctx context.Context, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	const maxLine = 4 * 1024 * 1024
	scanner.Buffer(make([]byte, 0, 64*1024), maxLine)
	encoder := json.NewEncoder(out)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		response := s.HandleJSONRPC(ctx, line)
		if response.ID == nil && response.Error == nil && response.Result == nil {
			continue
		}
		if err := encoder.Encode(response); err != nil {
			return fmt.Errorf("write mcp response: %w", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read mcp stdin: %w", err)
	}
	return nil
}

func (s *Server) HandleJSONRPC(ctx context.Context, body []byte) jsonRPCResponse {
	ctx, cancel := s.requestContext(ctx)
	defer cancel()
	finalize := func(response jsonRPCResponse) jsonRPCResponse {
		return s.capJSONRPCResponse(response)
	}
	if limit := s.maxJSONRPCRequestBytes(); limit > 0 && len(body) > limit {
		return finalize(jsonRPCResponse{JSONRPC: "2.0", Error: rpcErrorFrom(OperationError{
			Code:    "mcp_request_too_large",
			Message: "MCP request exceeded configured max_request_bytes",
		})})
	}

	var request jsonRPCRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return finalize(jsonRPCResponse{JSONRPC: "2.0", Error: rpcError(rpcParseError, "parse error", err.Error())})
	}
	if err := s.checkJSONRPCRateLimit(time.Now().UTC()); err != nil {
		return finalize(jsonRPCResponse{JSONRPC: "2.0", ID: request.ID, Error: rpcErrorFrom(err)})
	}
	if request.JSONRPC != "2.0" || request.Method == "" {
		return finalize(jsonRPCResponse{JSONRPC: "2.0", ID: request.ID, Error: rpcError(rpcInvalidRequest, "invalid request", nil)})
	}
	result, err := s.dispatchJSONRPC(ctx, request.Method, request.Params)
	if err != nil {
		return finalize(jsonRPCResponse{JSONRPC: "2.0", ID: request.ID, Error: rpcErrorFrom(err)})
	}
	return finalize(jsonRPCResponse{JSONRPC: "2.0", ID: request.ID, Result: result})
}

func (s *Server) dispatchJSONRPC(ctx context.Context, method string, params json.RawMessage) (any, error) {
	switch method {
	case "initialize":
		return map[string]any{
			"protocolVersion": ProtocolVersion,
			"serverInfo": map[string]string{
				"name":    "nivora-mcp",
				"version": "foundation",
			},
			"capabilities": map[string]any{
				"resources": map[string]any{},
				"tools":     map[string]any{},
				"prompts":   map[string]any{},
			},
		}, nil
	case "resources/list":
		resources, err := s.ListResources(ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{"resources": resources}, nil
	case "resources/read":
		var input struct {
			URI string `json:"uri"`
		}
		if err := decodeParams(params, &input); err != nil {
			return nil, err
		}
		if input.URI == "" {
			return nil, OperationError{Code: "mcp_invalid_arguments", Message: "uri is required"}
		}
		content, err := s.ReadResource(ctx, input.URI)
		if err != nil {
			return nil, err
		}
		return map[string]any{"contents": []ResourceContent{content}}, nil
	case "tools/list":
		tools, err := s.ListTools(ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{"tools": tools}, nil
	case "tools/call":
		var input struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := decodeParams(params, &input); err != nil {
			return nil, err
		}
		if input.Name == "" {
			return nil, OperationError{Code: "mcp_invalid_arguments", Message: "name is required"}
		}
		if input.Arguments == nil {
			input.Arguments = map[string]any{}
		}
		return s.CallTool(ctx, input.Name, input.Arguments)
	case "prompts/list":
		prompts, err := s.ListPrompts(ctx)
		if err != nil {
			return nil, err
		}
		return map[string]any{"prompts": prompts}, nil
	case "prompts/get":
		var input struct {
			Name      string            `json:"name"`
			Arguments map[string]string `json:"arguments"`
		}
		if err := decodeParams(params, &input); err != nil {
			return nil, err
		}
		if input.Name == "" {
			return nil, OperationError{Code: "mcp_invalid_arguments", Message: "name is required"}
		}
		if input.Arguments == nil {
			input.Arguments = map[string]string{}
		}
		return s.GetPrompt(ctx, input.Name, input.Arguments)
	default:
		return nil, OperationError{Code: "mcp_method_not_found", Message: "unknown MCP method " + method}
	}
}

func decodeParams(params json.RawMessage, out any) error {
	if len(params) == 0 || string(params) == "null" {
		return nil
	}
	if err := json.Unmarshal(params, out); err != nil {
		return OperationError{Code: "mcp_invalid_params", Message: err.Error()}
	}
	return nil
}

func rpcErrorFrom(err error) *jsonRPCError {
	var op OperationError
	if errors.As(err, &op) {
		code := rpcInternalError
		if op.Code == "mcp_method_not_found" {
			code = rpcMethodNotFound
		} else if op.Code == "mcp_invalid_arguments" || op.Code == "mcp_invalid_params" {
			code = rpcInvalidParams
		}
		return rpcError(code, op.Message, op)
	}
	return rpcError(rpcInternalError, err.Error(), nil)
}

func rpcError(code int, message string, data any) *jsonRPCError {
	return &jsonRPCError{Code: code, Message: message, Data: data}
}

func (s *Server) maxJSONRPCRequestBytes() int {
	return s.services.Config.MCP.MaxRequestBytes
}

func (s *Server) checkJSONRPCRateLimit(now time.Time) error {
	limit := s.services.Config.MCP.MaxRequestsPerMinute
	if limit <= 0 {
		return nil
	}
	s.rateLimitMu.Lock()
	defer s.rateLimitMu.Unlock()

	if s.rateLimitWindow.IsZero() || now.Sub(s.rateLimitWindow) >= time.Minute {
		s.rateLimitWindow = now
		s.rateLimitCount = 0
	}
	if s.rateLimitCount >= limit {
		return OperationError{Code: "mcp_rate_limited", Message: "MCP request rate limit exceeded"}
	}
	s.rateLimitCount++
	return nil
}

func (s *Server) capJSONRPCResponse(response jsonRPCResponse) jsonRPCResponse {
	limit := s.services.Config.MCP.MaxResponseBytes
	if limit <= 0 {
		return response
	}
	body, err := json.Marshal(response)
	if err != nil || len(body) <= limit {
		return response
	}
	capped := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      response.ID,
		Error: rpcError(rpcInternalError, "MCP response exceeded configured max_response_bytes", map[string]any{
			"code":               "mcp_response_too_large",
			"max_response_bytes": limit,
		}),
	}
	cappedBody, err := json.Marshal(capped)
	if err == nil && len(cappedBody) <= limit {
		return capped
	}
	return jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      response.ID,
		Error:   rpcError(rpcInternalError, "MCP response too large", nil),
	}
}
