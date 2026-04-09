package tools

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	analyticspb "github.com/KoiralaSam/ZorbaHealth/shared/proto/analytics"
)

type getHospitalAnalyticsInput struct {
	Auth string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

func RegisterGetHospitalAnalytics(s *mcp.Server, db *pgxpool.Pool, client analyticspb.AnalyticsServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_hospital_analytics",
		Description: "Get hospital analytics summary",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in getHospitalAnalyticsInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}

		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}

		if err := sharedauth.RequireActorType(claims, sharedauth.ActorStaff); err != nil {
			audit(db, claims, "get_hospital_analytics", "forbidden", err.Error())
			return errorResult(err.Error()), nil, nil
		}
		if !sharedauth.HasScope(claims, "hospital:analytics") {
			audit(db, claims, "get_hospital_analytics", "forbidden", "missing hospital:analytics")
			return errorResult("forbidden: missing hospital:analytics"), nil, nil
		}

		ctx = ctxWithForwardedToken(ctx, in.Auth)

		resp, err := client.GetHospitalSummary(ctx, &analyticspb.HospitalSummaryRequest{
			HospitalId: claims.HospitalID,
		})
		if err != nil {
			audit(db, claims, "get_hospital_analytics", "error", err.Error())
			return errorResult("analytics lookup failed"), nil, nil
		}

		payload, _ := json.Marshal(resp)
		audit(db, claims, "get_hospital_analytics", "success", "")
		return textResult(string(payload)), nil, nil
	})
}
