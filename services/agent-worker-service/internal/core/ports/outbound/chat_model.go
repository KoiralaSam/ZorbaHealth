package outbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/domain/models"
)

type ChatModel interface {
	Chat(ctx context.Context, systemPrompt string, messages []models.Message, tools []models.ToolDef) (*models.ChatResponse, error)
	BuildUnauthenticatedSystemPrompt(language, callerPhone string) string
	BuildAuthenticatedSystemPrompt(patientID, language, contextText string) string
	BuildPublicToolDefinitions() []models.ToolDef
	BuildAuthenticatedToolDefinitions() []models.ToolDef
}
