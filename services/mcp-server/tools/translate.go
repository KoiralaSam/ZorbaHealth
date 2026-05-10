package tools

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

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
			audit(db, claims, "translate", "forbidden", "forbidden: unsupported actor type")
			return errorResult("forbidden: unsupported actor type"), nil, nil
		}

		ctx = ctxWithForwardedToken(ctx, in.Auth)

		resp, err := client.Translate(ctx, &transpb.TranslateRequest{
			Text:       in.Text,
			TargetLang: in.TargetLang,
			SourceLang: in.SourceLang,
		})
		if err != nil {
			audit(db, claims, "translate", "error", err.Error())
			return errorResult(err.Error()), nil, nil
		}

		audit(db, claims, "translate", "success", "")
		return textResult(resp.GetTranslatedText()), nil, nil
	})
}
