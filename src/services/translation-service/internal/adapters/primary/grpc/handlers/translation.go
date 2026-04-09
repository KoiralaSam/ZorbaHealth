package handlers

import (
	"context"
	"errors"

	"github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/adapters/primary/grpc/mappers"
	domainerrors "github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/domain/errors"
	"github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/ports/inbound"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/translation"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type TranslationHandler struct {
	pb.UnimplementedTranslationServiceServer
	service inbound.TranslationService
}

func NewTranslationHandler(service inbound.TranslationService) *TranslationHandler {
	return &TranslationHandler{service: service}
}

func (h *TranslationHandler) Translate(ctx context.Context, req *pb.TranslateRequest) (*pb.TranslateResponse, error) {
	claims, err := sharedauth.ClaimsFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "no verified claims")
	}

	switch claims.ActorType {
	case sharedauth.ActorPatient, sharedauth.ActorStaff:
	default:
		return nil, status.Error(codes.PermissionDenied, "forbidden: unsupported actor type")
	}

	result, err := h.service.Translate(ctx, mappers.ProtoToRequest(req))
	if err != nil {
		return nil, mapDomainError(err)
	}

	return mappers.ResultToProto(result), nil
}

func mapDomainError(err error) error {
	switch {
	case errors.Is(err, domainerrors.ErrEmptyText),
		errors.Is(err, domainerrors.ErrInvalidLanguageCode),
		errors.Is(err, domainerrors.ErrTextTooLong),
		errors.Is(err, domainerrors.ErrUnsupportedLanguage):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, domainerrors.ErrProviderUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	case errors.Is(err, domainerrors.ErrTranslationFailed):
		return status.Error(codes.Internal, err.Error())
	default:
		return status.Errorf(codes.Internal, "unexpected error: %v", err)
	}
}
