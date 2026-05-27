package tools

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	healthpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_records"
)

type answerHealthQuestionInput struct {
	Question string  `json:"question" jsonschema:"patient health record question"`
	TopK     float64 `json:"topK,omitempty" jsonschema:"optional result count"`
	Auth     string  `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

func RegisterAnswerHealthQuestion(s *mcp.Server, db *pgxpool.Pool, client healthpb.HealthRecordServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "answer_health_question",
		Description: "Answer a patient's question using only their verified health records",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in answerHealthQuestionInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}

		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}
		if err := sharedauth.RequireActorType(claims, sharedauth.ActorPatient); err != nil {
			auditCompat(db, claims, "answer_health_question", "forbidden", err.Error())
			return errorResult(err.Error()), nil, nil
		}
		if !sharedauth.HasScope(claims, "records:read") {
			auditCompat(db, claims, "answer_health_question", "forbidden", "missing records:read")
			return errorResult("forbidden: missing records:read"), nil, nil
		}

		topK := int32(5)
		if in.TopK > 0 {
			topK = int32(in.TopK)
		}

		correlationID := auditStart(ctx, claims, sharedaudit.EventHealthRecordSummarized, "answer_health_question", map[string]any{
			"top_k": topK,
		})
		ctx = ctxWithForwardedToken(ctx, in.Auth)
		resp, err := client.AnswerPatientQuestion(ctx, &healthpb.AnswerPatientQuestionRequest{
			PatientId: claims.PatientID,
			Question:  in.Question,
			TopK:      topK,
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventHealthRecordSummarized, "answer_health_question", "error", err.Error(), correlationID, nil)
			return errorResult("question answering failed"), nil, nil
		}

		payload, _ := json.Marshal(map[string]any{
			"answer":    resp.GetAnswer(),
			"citations": resp.GetCitations(),
		})
		auditComplete(ctx, db, claims, sharedaudit.EventHealthRecordSummarized, "answer_health_question", "success", "", correlationID, map[string]any{
			"citation_count": len(resp.GetCitations()),
		})
		return textResult(string(payload)), nil, nil
	})
}
