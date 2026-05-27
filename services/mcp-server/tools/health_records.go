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
			auditCompat(db, claims, "search_health_records", "forbidden", err.Error())
			return errorResult(err.Error()), nil, nil
		}
		if !sharedauth.HasScope(claims, "records:read") {
			auditCompat(db, claims, "search_health_records", "forbidden", "missing records:read")
			return errorResult("forbidden: missing records:read"), nil, nil
		}
		topK := int32(5)
		if in.TopK > 0 {
			topK = int32(in.TopK)
		}

		correlationID := auditStart(ctx, claims, sharedaudit.EventHealthRecordSearched, "search_health_records", map[string]any{
			"top_k": topK,
		})
		ctx = ctxWithForwardedToken(ctx, in.Auth)

		resp, err := client.SearchRecords(ctx, &healthpb.SearchRequest{
			PatientId: claims.PatientID,
			Query:     in.Query,
			TopK:      topK,
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventHealthRecordSearched, "search_health_records", "error", err.Error(), correlationID, nil)
			return errorResult("search failed"), nil, nil
		}

		type citation struct {
			ChunkID          string  `json:"chunk_id"`
			RecordID         string  `json:"record_id"`
			FhirResourceType string  `json:"fhir_resource_type"`
			SourceFile       string  `json:"source_file"`
			Score            float32 `json:"score"`
			Excerpt          string  `json:"excerpt"`
		}
		out := make([]citation, 0, len(resp.GetChunks()))
		for _, ch := range resp.GetChunks() {
			excerpt := ch.GetText()
			if len(excerpt) > 240 {
				excerpt = excerpt[:240] + "..."
			}
			out = append(out, citation{
				ChunkID:          ch.GetChunkId(),
				RecordID:         ch.GetRecordId(),
				FhirResourceType: ch.GetFhirResourceType(),
				SourceFile:       ch.GetSourceFile(),
				Score:            ch.GetScore(),
				Excerpt:          excerpt,
			})
		}
		payload, _ := json.Marshal(out)
		auditComplete(ctx, db, claims, sharedaudit.EventHealthRecordSearched, "search_health_records", "success", "", correlationID, map[string]any{
			"chunk_count": len(resp.GetChunks()),
		})
		return textResult(string(payload)), nil, nil
	})
}
