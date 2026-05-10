package tools

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	healthpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_records"
)

type searchHealthRecordsInput struct {
	Query string  `json:"query" jsonschema:"search query"`
	TopK  float64 `json:"topK,omitempty" jsonschema:"optional result count"`
	Auth  string  `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

func RegisterSearchHealthRecords(s *mcp.Server, db *pgxpool.Pool, client healthpb.HealthRecordServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "search_health_records",
		Description: "Search the patient's own health records",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in searchHealthRecordsInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}

		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}

		if err := sharedauth.RequireActorType(claims, sharedauth.ActorPatient); err != nil {
			audit(db, claims, "search_health_records", "forbidden", err.Error())
			return errorResult(err.Error()), nil, nil
		}
		if !sharedauth.HasScope(claims, "records:read") {
			audit(db, claims, "search_health_records", "forbidden", "missing records:read")
			return errorResult("forbidden: missing records:read"), nil, nil
		}

		topK := int32(5)
		if in.TopK > 0 {
			topK = int32(in.TopK)
		}

		ctx = ctxWithForwardedToken(ctx, in.Auth)

		resp, err := client.SearchRecords(ctx, &healthpb.SearchRequest{
			PatientId: claims.PatientID,
			Query:     in.Query,
			TopK:      topK,
		})
		if err != nil {
			audit(db, claims, "search_health_records", "error", err.Error())
			return errorResult("search failed"), nil, nil
		}

		payload, _ := json.Marshal(resp.GetChunks())
		audit(db, claims, "search_health_records", "success", "")
		return textResult(string(payload)), nil, nil
	})
}
