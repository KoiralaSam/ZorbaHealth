package handlers

import (
	"errors"
	"testing"

	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/appointment-service/internal/core/domain/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapErrorCodes(t *testing.T) {
	cases := []struct {
		err  error
		code codes.Code
	}{
		{domainerrors.ErrNotFound, codes.NotFound},
		{domainerrors.ErrForbidden, codes.PermissionDenied},
		{domainerrors.ErrConsentRequired, codes.PermissionDenied},
		{domainerrors.ErrUnauthorized, codes.Unauthenticated},
		{domainerrors.ErrConflict, codes.FailedPrecondition},
		{domainerrors.ErrSlotUnavailable, codes.FailedPrecondition},
		{domainerrors.ErrInvalidArgument, codes.InvalidArgument},
		{errors.New("boom"), codes.Internal},
	}
	for _, tc := range cases {
		st, ok := status.FromError(mapError(tc.err))
		if !ok {
			t.Fatalf("expected status error for %v", tc.err)
		}
		if st.Code() != tc.code {
			t.Fatalf("for %v expected %v got %v", tc.err, tc.code, st.Code())
		}
	}
}
