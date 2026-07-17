package livekit

import (
	"context"
	"testing"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/ports/outbound"
)

func TestStubWelfareCallProviderUsesStableRoom(t *testing.T) {
	provider := &stubWelfareCallProvider{}
	result, err := provider.StartWelfareCheckCall(context.Background(), outbound.WelfareCheckCallInput{
		RunID:        "run-123",
		RoomName:     "welfare-check-run-123",
		PatientPhone: "+15551234567",
		PatientToken: "secret-token",
		ScheduledAt:  time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.RoomName != "welfare-check-run-123" {
		t.Fatalf("room=%s", result.RoomName)
	}
}

func TestNewWelfareCheckCallProviderRequiresExplicitStub(t *testing.T) {
	t.Setenv("LIVEKIT_USE_STUB", "true")
	provider := NewWelfareCheckCallProvider()
	if _, ok := provider.(*stubWelfareCallProvider); !ok {
		t.Fatalf("expected stub provider when LIVEKIT_USE_STUB=true")
	}
}
