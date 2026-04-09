package tools

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	locpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/location"
)

type getLocationInput struct {
	SessionID string `json:"sessionID" jsonschema:"patient session ID"`
	Auth      string `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

type findNearestHospitalInput struct {
	Lat       float64 `json:"lat" jsonschema:"latitude"`
	Lng       float64 `json:"lng" jsonschema:"longitude"`
	PlaceType string  `json:"placeType,omitempty" jsonschema:"hospital, urgent_care, or pharmacy"`
	Auth      string  `json:"_auth" jsonschema:"bearer JWT" jsonschema_extras:"required=true"`
}

func RegisterGetLocation(s *mcp.Server, db *pgxpool.Pool, client locpb.LocationServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "get_location",
		Description: "Get the patient's current location",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in getLocationInput) (*mcp.CallToolResult, any, error) {
		if err := requireToken(in.Auth); err != nil {
			return errorResult("unauthorized"), nil, nil
		}

		claims, err := verifyToken(in.Auth)
		if err != nil {
			return errorResult("unauthorized"), nil, nil
		}

		if err := sharedauth.RequireActorType(claims, sharedauth.ActorPatient); err != nil {
			audit(db, claims, "get_location", "forbidden", err.Error())
			return errorResult(err.Error()), nil, nil
		}
		if !sharedauth.HasScope(claims, "location:read") {
			audit(db, claims, "get_location", "forbidden", "missing location:read")
			return errorResult("forbidden: missing location:read"), nil, nil
		}
		if claims.SessionID != "" && in.SessionID != claims.SessionID {
			msg := fmt.Sprintf("session_id mismatch: token=%q request=%q", claims.SessionID, in.SessionID)
			audit(db, claims, "get_location", "forbidden", msg)
			return errorResult(msg), nil, nil
		}

		ctx = ctxWithForwardedToken(ctx, in.Auth)

		resp, err := client.GetLocation(ctx, &locpb.GetLocationRequest{
			SessionId: in.SessionID,
		})
		if err != nil {
			audit(db, claims, "get_location", "error", err.Error())
			return errorResult("location lookup failed"), nil, nil
		}

		out := fmt.Sprintf(`{"lat":%f,"lng":%f,"method":"%s","accuracy":"%s"}`,
			resp.GetLat(), resp.GetLng(), resp.GetMethod(), resp.GetAccuracy())

		audit(db, claims, "get_location", "success", "")
		return textResult(out), nil, nil
	})
}

func RegisterFindNearestHospital(s *mcp.Server, db *pgxpool.Pool, client locpb.LocationServiceClient) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "find_nearest_hospital",
		Description: "Find the nearest hospital or care facility",
	}, func(ctx context.Context, req *mcp.CallToolRequest, in findNearestHospitalInput) (*mcp.CallToolResult, any, error) {
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
			audit(db, claims, "find_nearest_hospital", "forbidden", "forbidden: unsupported actor type")
			return errorResult("forbidden: unsupported actor type"), nil, nil
		}

		placeType := in.PlaceType
		if placeType == "" {
			placeType = "hospital"
		}

		ctx = ctxWithForwardedToken(ctx, in.Auth)

		resp, err := client.FindNearestHospital(ctx, &locpb.FindHospitalRequest{
			Lat:       in.Lat,
			Lng:       in.Lng,
			PlaceType: placeType,
		})
		if err != nil {
			audit(db, claims, "find_nearest_hospital", "error", err.Error())
			return errorResult("hospital lookup failed"), nil, nil
		}

		out := fmt.Sprintf(`{"name":%q,"address":%q,"directions_url":%q,"phone":%q}`,
			resp.GetName(), resp.GetAddress(), resp.GetDirectionsUrl(), resp.GetPhone())

		audit(db, claims, "find_nearest_hospital", "success", "")
		return textResult(out), nil, nil
	})
}
