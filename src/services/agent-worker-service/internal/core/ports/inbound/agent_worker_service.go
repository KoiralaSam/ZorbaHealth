package inbound

import (
	"context"

	"github.com/KoiralaSam/ZorbaHealth/services/agent-worker-service/internal/core/domain/models"
)

type AgentWorkerService interface {
	StartSession(ctx context.Context, start models.SessionStart) error
}
