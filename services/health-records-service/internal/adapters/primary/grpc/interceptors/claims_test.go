package interceptors

import (
	"context"
	"testing"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	healthpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_records"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestClaimsInterceptorInjectsForwardedTokenForDownstreamClients(t *testing.T) {
	const secret = "test-patient-secret"
	t.Setenv("PATIENT_SERVICE_JWT_SECRET", secret)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"actorType": "patient",
		"patientID": "patient-1",
		"sessionID": "session-1",
		"exp":       time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-forwarded-token", signed))
	_, err = ClaimsInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		forwarded, ok := grpcclient.ForwardedTokenFromContext(ctx)
		if !ok {
			t.Fatal("forwarded token was not injected into context")
		}
		if forwarded != signed {
			t.Fatalf("forwarded token = %q, want signed token", forwarded)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
}

func TestClaimsInterceptorNormalizesBearerForwardedToken(t *testing.T) {
	const secret = "test-staff-secret"
	t.Setenv("AUTH_SERVICE_JWT_SECRET", secret)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"actorType":  "staff",
		"staffID":    "staff-1",
		"hospitalID": "hospital-1",
		"role":       "doctor",
		"exp":        time.Now().Add(time.Hour).Unix(),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-forwarded-token", "Bearer "+signed))
	_, err = ClaimsInterceptor(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		forwarded, ok := grpcclient.ForwardedTokenFromContext(ctx)
		if !ok {
			t.Fatal("forwarded token was not injected into context")
		}
		if forwarded != signed {
			t.Fatalf("forwarded token = %q, want normalized signed token", forwarded)
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
}

func TestChainAllowsIngestFHIRBundleWithInternalTokenOnly(t *testing.T) {
	t.Setenv("INTERNAL_SERVICE_SECRET", "test-internal-secret")

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-internal-token", "test-internal-secret"))
	called := false
	_, err := Chain()(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: healthpb.HealthRecordService_IngestFHIRBundle_FullMethodName,
	}, func(ctx context.Context, req any) (any, error) {
		called = true
		return nil, nil
	})
	if err != nil {
		t.Fatalf("interceptor: %v", err)
	}
	if !called {
		t.Fatal("handler was not called")
	}
}

func TestChainRequiresForwardedTokenForSearchRecords(t *testing.T) {
	t.Setenv("INTERNAL_SERVICE_SECRET", "test-internal-secret")

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("x-internal-token", "test-internal-secret"))
	_, err := Chain()(ctx, nil, &grpc.UnaryServerInfo{
		FullMethod: healthpb.HealthRecordService_SearchRecords_FullMethodName,
	}, func(ctx context.Context, req any) (any, error) {
		t.Fatal("handler should not be called")
		return nil, nil
	})
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("status.Code(err) = %v, want %v (err=%v)", status.Code(err), codes.Unauthenticated, err)
	}
}
