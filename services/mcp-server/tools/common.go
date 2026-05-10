package tools

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
)

func verifyToken(token string) (*sharedauth.Claims, error) {
	return sharedauth.VerifyToken(token)
}

func ctxWithForwardedToken(ctx context.Context, token string) context.Context {
	return grpcclient.WithForwardedToken(ctx, token)
}

func audit(db *pgxpool.Pool, claims *sharedauth.Claims, tool, outcome, errMsg string) {
	sharedauth.LogAuditEventAsync(db, claims, tool, outcome, errMsg)
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

func errorResult(msg string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: msg},
		},
	}
}

func requireToken(token string) error {
	if token == "" {
		return fmt.Errorf("missing _auth")
	}
	return nil
}
