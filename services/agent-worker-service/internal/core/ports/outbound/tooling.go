package outbound

import "context"

type ToolCaller interface {
	CallTool(ctx context.Context, name string, args map[string]any) (string, error)
}

type SessionTokenIssuer interface {
	MintSessionToken(patientID, sessionID string, scopes []string) (string, error)
}
