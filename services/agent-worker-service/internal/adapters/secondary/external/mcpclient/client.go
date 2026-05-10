package mcpclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/ports/outbound"
)

type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
	nextID atomic.Int64
}

var _ outbound.ToolCaller = (*Client)(nil)

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int64  `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func New(binaryPath string) (*Client, error) {
	if strings.TrimSpace(binaryPath) == "" {
		return nil, fmt.Errorf("mcp binary path is required")
	}

	cmd := exec.Command(binaryPath)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start mcp-server: %w", err)
	}

	c := &Client{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}

	if _, err := c.rpc(context.Background(), "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "zorba-agent-worker",
			"version": "1.0.0",
		},
	}); err != nil {
		return nil, fmt.Errorf("mcp initialize: %w", err)
	}

	if err := c.notify("notifications/initialized", map[string]any{}); err != nil {
		return nil, fmt.Errorf("mcp initialized notification: %w", err)
	}

	return c, nil
}

func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (string, error) {
	raw, err := c.rpc(ctx, "tools/call", map[string]any{
		"name":      name,
		"arguments": args,
	})
	if err != nil {
		return "", err
	}

	var parsed struct {
		IsError bool `json:"isError"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("parse tool result: %w", err)
	}

	var parts []string
	for _, item := range parsed.Content {
		if item.Type == "text" && item.Text != "" {
			parts = append(parts, item.Text)
		}
	}
	text := strings.Join(parts, "\n")
	if parsed.IsError {
		return "", fmt.Errorf("tool error: %s", text)
	}
	return text, nil
}

func (c *Client) Close() error {
	_ = c.stdin.Close()
	if c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
	}
	return c.cmd.Wait()
}

func (c *Client) notify(method string, params any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	if _, err := c.stdin.Write(data); err != nil {
		return fmt.Errorf("write mcp notification: %w", err)
	}
	return nil
}

func (c *Client) rpc(_ context.Context, method string, params any) (json.RawMessage, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.nextID.Add(1)
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')

	if _, err := c.stdin.Write(data); err != nil {
		return nil, fmt.Errorf("write mcp request: %w", err)
	}

	for {
		line, err := c.stdout.ReadBytes('\n')
		if err != nil {
			return nil, fmt.Errorf("read mcp response: %w", err)
		}

		var resp jsonRPCResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}
		if resp.ID != id {
			continue
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp.Result, nil
	}
}
