package mappers

import (
	"github.com/KoiralaSam/ZorbaHealth/services/translation-service/internal/core/domain/models"
	pb "github.com/KoiralaSam/ZorbaHealth/shared/proto/translation"
)

func ProtoToRequest(req *pb.TranslateRequest) models.TranslationRequest {
	return models.TranslationRequest{
		Text:       req.GetText(),
		TargetLang: req.GetTargetLang(),
		SourceLang: req.GetSourceLang(),
	}
}

func ResultToProto(result *models.TranslationResult) *pb.TranslateResponse {
	return &pb.TranslateResponse{
		TranslatedText: result.TranslatedText,
		DetectedLang:   result.DetectedLang,
	}
}
