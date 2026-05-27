package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	locpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/location"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
			auditCompat(db, claims, "get_location", "forbidden", err.Error())
			return errorResult(err.Error()), nil, nil
		}
		if !sharedauth.HasScope(claims, "location:read") {
			auditCompat(db, claims, "get_location", "forbidden", "missing location:read")
			return errorResult("forbidden: missing location:read"), nil, nil
		}
		if claims.SessionID != "" && in.SessionID != claims.SessionID {
			msg := fmt.Sprintf("session_id mismatch: token=%q request=%q", claims.SessionID, in.SessionID)
			auditCompat(db, claims, "get_location", "forbidden", msg)
			return errorResult(msg), nil, nil
		}
		if strings.HasPrefix(claims.PatientID, "session:") {
			auditCompat(db, claims, "get_location", "forbidden", "unverified session")
			return errorResult("verify your phone number before sharing location"), nil, nil
		}
		// Portal grants use global (empty) scope; this call is already bound to the voice session via JWT sessionID above.
		allowed, denialReason, err := checkConsent(ctx, db, in.Auth, claims.PatientID, sharedaudit.ConsentLocationAccess, "")
		if err != nil {
			auditCompat(db, claims, "get_location", "error", err.Error())
			return errorResult("consent check failed"), nil, nil
		}
		if !allowed {
			auditCompat(db, claims, "get_location", "consent-denied", denialReason)
			return errorResult(denialReason), nil, nil
		}

		correlationID := auditStart(ctx, claims, sharedaudit.EventLocationRequested, "get_location", map[string]any{
			"session_id": in.SessionID,
		})
		ctx = ctxWithForwardedToken(ctx, in.Auth)

		resp, err := client.GetLocation(ctx, &locpb.GetLocationRequest{
			SessionId: in.SessionID,
		})
		if err != nil {
			if st, ok := status.FromError(err); ok && st.Code() == codes.NotFound {
				auditComplete(ctx, db, claims, sharedaudit.EventLocationRequested, "get_location", "not_found", st.Message(), correlationID, nil)
				return textResult(`{"available":false,"reason":"no_location","message":"No location is stored for this voice session yet. The patient can open the Zorba app with location sharing enabled during the call, or you can continue without GPS."}`), nil, nil
			}
			auditComplete(ctx, db, claims, sharedaudit.EventLocationRequested, "get_location", "error", err.Error(), correlationID, nil)
			return errorResult("location lookup failed"), nil, nil
		}

		out := fmt.Sprintf(`{"lat":%f,"lng":%f,"method":"%s","accuracy":"%s"}`,
			resp.GetLat(), resp.GetLng(), resp.GetMethod(), resp.GetAccuracy())

		auditComplete(ctx, db, claims, sharedaudit.EventLocationRequested, "get_location", "success", "", correlationID, nil)
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
			auditCompat(db, claims, "find_nearest_hospital", "forbidden", "forbidden: unsupported actor type")
			return errorResult("forbidden: unsupported actor type"), nil, nil
		}

		placeType := in.PlaceType
		if placeType == "" {
			placeType = "hospital"
		}

		correlationID := auditStart(ctx, claims, sharedaudit.EventLocationRequested, "find_nearest_hospital", map[string]any{
			"place_type": placeType,
		})
		ctx = ctxWithForwardedToken(ctx, in.Auth)

		resp, err := client.FindNearestHospital(ctx, &locpb.FindHospitalRequest{
			Lat:       in.Lat,
			Lng:       in.Lng,
			PlaceType: placeType,
		})
		if err != nil {
			auditComplete(ctx, db, claims, sharedaudit.EventLocationRequested, "find_nearest_hospital", "error", err.Error(), correlationID, nil)
			return errorResult("hospital lookup failed"), nil, nil
		}

		out := fmt.Sprintf(`{"name":%q,"address":%q,"directions_url":%q,"phone":%q}`,
			resp.GetName(), resp.GetAddress(), resp.GetDirectionsUrl(), resp.GetPhone())

		auditComplete(ctx, db, claims, sharedaudit.EventLocationRequested, "find_nearest_hospital", "success", "", correlationID, nil)
		return textResult(out), nil, nil
	})
}
