package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/domain/models"
	"github.com/KoiralaSam/ZorbaHealth/services/patient-service/internal/core/services"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
)

type HttpHandler struct {
	Service *services.PatientService
}

func (h *HttpHandler) PatientRegisterHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody *models.RegisterPatientRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	defer r.Body.Close()

	ctx := r.Context()

	_, err := h.Service.StartRegistrationWithVerification(ctx, reqBody)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to start registration: " + err.Error(),
		})
		return
	}

	writeJson(w, http.StatusOK, map[string]string{"message": "Verification email sent. Please check your inbox."}, nil)
}

func (h *HttpHandler) PatientLoginHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody *models.Patient
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	defer r.Body.Close()

	ctx := r.Context()

	patient, err := h.Service.LoginPatient(ctx, reqBody)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to register patient: " + err.Error(),
		})
		return
	}

	writeJson(w, http.StatusOK, patient, nil)

}
