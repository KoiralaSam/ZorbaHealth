package main

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/KoiralaSam/ZorbaHealth/services/api-gateway/cmd/api-gateway/grpc_clients"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/registration_verification"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// corsMiddleware adds CORS headers to allow requests from the web frontend
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Allow requests from the Next.js frontend (localhost:3000)
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// PatientLoginHandler handles patient authentication
func PatientLoginHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody PatientLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	defer r.Body.Close()

	//validation
	if reqBody.PhoneNumber == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Phone number is required",
		})
		return
	}

	// TODO: call patient service to initiate login
	patientAuthServiceClient, err := grpc_clients.NewPatientAuthServiceClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to create auth client: " + err.Error(),
		})
		return
	}

	defer patientAuthServiceClient.Close()

	writeJson(w, http.StatusOK, nil, nil)
}

func PatientRegisterHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody PatientRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	defer r.Body.Close()

	//validation
	if reqBody.PhoneNumber == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Phone number is required",
		})
		return
	}

	patientAuthServiceClient, err := grpc_clients.NewPatientAuthServiceClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to create auth client: " + err.Error(),
		})
		return
	}
	defer patientAuthServiceClient.Close()

	var dateOfBirth *timestamppb.Timestamp
	if !reqBody.DateOfBirth.IsZero() {
		dateOfBirth = timestamppb.New(reqBody.DateOfBirth)
	}
	response, err := patientAuthServiceClient.RegistrationClient.StartRegistration(context.Background(), &registration_verification.StartRegistrationRequest{
		PhoneNumber:  reqBody.PhoneNumber,
		Email:       reqBody.Email,
		Password:    reqBody.Password,
		FullName:    reqBody.FullName,
		DateOfBirth: dateOfBirth,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.AlreadyExists {
			writeJson(w, http.StatusConflict, nil, &contracts.APIError{
				Code:    "ALREADY_EXISTS",
				Message: st.Message(),
			})
			return
		}
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to start registration: " + err.Error(),
		})
		return
	}

	writeJson(w, http.StatusOK, response, nil)
}

// PatientRegisterVerifyHandler handles email verification (step 2 of registration).
func PatientRegisterVerifyHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	defer r.Body.Close()

	if reqBody.Token == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "token is required",
		})
		return
	}

	client, err := grpc_clients.NewPatientAuthServiceClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to create auth client: " + err.Error(),
		})
		return
	}
	defer client.Close()

	response, err := client.RegistrationClient.VerifyEmail(context.Background(), &registration_verification.VerifyEmailRequest{Token: reqBody.Token})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "VERIFICATION_FAILED",
			Message: "Invalid or expired verification link: " + err.Error(),
		})
		return
	}

	writeJson(w, http.StatusOK, response, nil)
}

// PatientRegisterVerifyOTPHandler verifies the OTP sent to the patient's phone (step 1 of registration).
func PatientRegisterVerifyOTPHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody PatientRegisterVerifyOTPRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	defer r.Body.Close()

	if reqBody.PhoneNumber == "" || reqBody.OTP == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "phone_number and otp are required",
		})
		return
	}

	client, err := grpc_clients.NewPatientAuthServiceClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to create auth client: " + err.Error(),
		})
		return
	}
	defer client.Close()

	response, err := client.RegistrationClient.VerifyPhoneOTP(context.Background(), &registration_verification.VerifyPhoneOTPRequest{
		PhoneNumber: reqBody.PhoneNumber,
		Otp:         reqBody.OTP,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "VERIFICATION_FAILED",
			Message: "Invalid or expired OTP: " + err.Error(),
		})
		return
	}

	writeJson(w, http.StatusOK, response, nil)
}

// HospitalLoginHandler handles hospital staff authentication
func HospitalLoginHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody HospitalLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	defer r.Body.Close()

	// TODO: Implement actual authentication logic
	response := HospitalLoginResponse{
		Message: "Hospital login endpoint - Implementation pending",
	}
	writeJson(w, http.StatusOK, response, nil)
}
