package tools

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	transpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/translation"
)

type translateInput struct {
	Text       string `json:"text" jsonschema:"text to translate"`
	TargetLang string `json:"targetLang" jsonschema:"target ISO 639-1 language code"`
	SourceLang string `json:"sourceLang,omitempty" jsonschema:"optional source ISO 639-1 language code"`
	Auth       string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

func RegisterTranslate(s *mcp.Server, db *pgxpool.Pool, client transpb.TranslationServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "translate",
		Description: "Translate text to another language",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in translateInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}

		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}

		switch claims.ActorType {
		case sharedauth.ActorPatient, sharedauth.ActorStaff:
		default:
			auditCompat(db, claims, "translate", "forbidden", "forbidden: unsupported actor type")
			return errorResult("forbidden: unsupported actor type"), nil, nil
		}

		correlationID := auditStart(ctx, claims, sharedaudit.EventTranslationRequested, "translate", map[string]any{
			"target_lang": in.TargetLang,
		})
		ctx = ctxWithForwardedToken(ctx, in.Auth)

		resp, err := client.Translate(ctx, &transpb.TranslateRequest{
			Text:       in.Text,
			TargetLang: in.TargetLang,
			SourceLang: in.SourceLang,
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventTranslationRequested, "translate", "error", err.Error(), correlationID, nil)
			return errorResult(err.Error()), nil, nil
		}

		meta := map[string]any{
			"provider":         resp.GetTranslationProvider(),
			"confidence_score": resp.GetConfidenceScore(),
		}
		if resp.GetMedicalTermPreservationCheck() {
			meta["medical_term_preservation_check"] = true
		}
		auditComplete(ctx, db, claims, sharedaudit.EventTranslationRequested, "translate", "success", "", correlationID, meta)
		out := resp.GetTranslatedText()
		if resp.GetAdvisoryMessage() != "" {
			out += "\n\n" + resp.GetAdvisoryMessage()
		}
		return textResult(out), nil, nil
	})
}
