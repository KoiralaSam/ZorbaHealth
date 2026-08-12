package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/api-gateway/cmd/api-gateway/grpc_clients"
	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/grpcclient"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
	auditportalpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/auditportal"
	authpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/auth"
	healthpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_records"
	patientpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient"
	"github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/registration_verification"
	schedpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patient/scheduling"
	patientportalpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/patientportal"
	"github.com/KoiralaSam/ZorbaHealth/shared/tracing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type corsConfig struct {
	AllowedOrigins map[string]struct{}
	// AllowLocalhost permits any http(s) origin on localhost/127.0.0.1/[::1],
	// regardless of port. Meant for local development only, where verification
	// tooling serves the web app from ephemeral ports.
	AllowLocalhost bool
	AllowedMethods string
	AllowedHeaders string
}

func newCORSConfigFromEnv() corsConfig {
	rawOrigins := env.GetString("API_GATEWAY_ALLOWED_ORIGINS", "http://localhost:3000")
	origins := map[string]struct{}{}
	for _, origin := range strings.Split(rawOrigins, ",") {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		origins[origin] = struct{}{}
	}

	return corsConfig{
		AllowedOrigins: origins,
		AllowLocalhost: env.GetBool("API_GATEWAY_ALLOW_LOCALHOST_ORIGINS", false),
		AllowedMethods: "GET, POST, PUT, DELETE, OPTIONS",
		AllowedHeaders: "Content-Type, Authorization, X-Zorba-Client",
	}
}

var gatewayCORS = newCORSConfigFromEnv()

func (c corsConfig) isAllowedOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	if _, ok := c.AllowedOrigins[origin]; ok {
		return true
	}
	return c.AllowLocalhost && isLocalhostOrigin(origin)
}

func isLocalhostOrigin(origin string) bool {
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	host := parsed.Hostname()
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func (c corsConfig) apply(w http.ResponseWriter, origin string) bool {
	w.Header().Add("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Methods", c.AllowedMethods)
	w.Header().Set("Access-Control-Allow-Headers", c.AllowedHeaders)

	if !c.isAllowedOrigin(origin) {
		return false
	}

	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	return true
}

// corsMiddleware adds CORS headers for explicitly trusted web and mobile origins.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		originAllowed := gatewayCORS.apply(w, origin)

		// Handle preflight OPTIONS request
		if r.Method == http.MethodOptions {
			if origin != "" && !originAllowed {
				writeJson(w, http.StatusForbidden, nil, &contracts.APIError{
					Code:    "FORBIDDEN",
					Message: "Origin is not allowed",
				})
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

var tracer = tracing.GetTracer("api-gateway")

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

	email, phoneNumber := patientLoginIdentifiers(reqBody)
	if email == "" && phoneNumber == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Email or phone number is required",
		})
		return
	}
	identifier := email
	if identifier == "" {
		identifier = phoneNumber
	}
	if !allowLoginRate(r, "patient", identifier) {
		writeRateLimited(w)
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

	response, err := patientAuthServiceClient.LoginClient.Login(r.Context(), &patientpb.LoginRequest{
		PhoneNumber: phoneNumber,
		Email:       email,
		Password:    reqBody.Password,
	})
	if err != nil {
		writeJson(w, http.StatusUnauthorized, nil, &contracts.APIError{
			Code:    "UNAUTHORIZED",
			Message: "Patient login failed: " + err.Error(),
		})
		return
	}
	if refresh := response.GetRefreshToken(); refresh != "" {
		setRefreshCookie(w, cookieNamePatientRefresh, refresh, cookiePathPatientRefresh)
	}
	out := PatientLoginResponse{
		Message:     response.GetMessage(),
		AccessToken: response.GetAccessToken(),
		PatientID:   response.GetPatientID(),
	}
	if clientKindFromRequest(r) == "mobile" {
		out.RefreshToken = response.GetRefreshToken()
	}
	writeJson(w, http.StatusOK, out, nil)
}

func PatientRegisterHandler(w http.ResponseWriter, r *http.Request) {
	ctx, span := tracer.Start(r.Context(), "PatientLoginHandler")
	defer span.End()

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
	if !allowOTPSendRate(r, reqBody.PhoneNumber) {
		writeRateLimited(w)
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
	response, err := patientAuthServiceClient.RegistrationClient.StartRegistration(ctx, &registration_verification.StartRegistrationRequest{
		PhoneNumber: reqBody.PhoneNumber,
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
	ctx, span := tracer.Start(r.Context(), "PatientLoginHandler")
	defer span.End()

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

	response, err := client.RegistrationClient.VerifyEmail(ctx, &registration_verification.VerifyEmailRequest{Token: reqBody.Token})
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
	if !allowOTPVerifyRate(r, reqBody.PhoneNumber) {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "VERIFICATION_FAILED",
			Message: "Invalid or expired OTP",
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
		recordOTPVerifyFailure(r, reqBody.PhoneNumber)
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "VERIFICATION_FAILED",
			Message: "Invalid or expired OTP",
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

	if strings.TrimSpace(reqBody.Email) == "" || strings.TrimSpace(reqBody.Password) == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "email and password are required",
		})
		return
	}
	email := strings.TrimSpace(reqBody.Email)
	if !allowLoginRate(r, "staff", email) {
		writeRateLimited(w)
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

	response, err := client.HospitalLoginClient.Login(r.Context(), &authpb.LoginRequest{
		Email:    email,
		Password: reqBody.Password,
	})
	if err != nil {
		writeJson(w, http.StatusUnauthorized, nil, &contracts.APIError{
			Code:    "UNAUTHORIZED",
			Message: "Hospital login failed: " + err.Error(),
		})
		return
	}

	claims, err := sharedauth.VerifyToken(response.GetAccessToken())
	if err != nil || claims.ActorType != sharedauth.ActorStaff || strings.TrimSpace(claims.HospitalID) == "" {
		writeJson(w, http.StatusForbidden, nil, &contracts.APIError{
			Code:    "FORBIDDEN",
			Message: "Hospital staff access is required",
		})
		return
	}

	if refresh := response.GetRefreshToken(); refresh != "" {
		setRefreshCookie(w, cookieNameHospitalRefresh, refresh, cookiePathHospitalRefresh)
	}
	out := HospitalLoginResponse{
		Message:     response.GetMessage(),
		AccessToken: response.GetAccessToken(),
		HospitalID:  claims.HospitalID,
		StaffID:     claims.StaffID,
		Role:        claims.Role,
	}
	if clientKindFromRequest(r) == "mobile" {
		out.RefreshToken = response.GetRefreshToken()
	}
	writeJson(w, http.StatusOK, out, nil)
}

func HospitalPatientSummaryHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody HospitalPatientSummaryRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	defer r.Body.Close()

	accessToken := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	if accessToken == "" {
		writeJson(w, http.StatusUnauthorized, nil, &contracts.APIError{
			Code:    "UNAUTHORIZED",
			Message: "Authorization bearer token is required",
		})
		return
	}
	if strings.TrimSpace(reqBody.PatientID) == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "patient_id, name, or email is required",
		})
		return
	}

	claims, err := sharedauth.VerifyToken(accessToken)
	if err != nil {
		writeJson(w, http.StatusUnauthorized, nil, &contracts.APIError{
			Code:    "UNAUTHORIZED",
			Message: "Invalid hospital token",
		})
		return
	}
	if claims.ActorType != sharedauth.ActorStaff || strings.TrimSpace(claims.HospitalID) == "" {
		writeJson(w, http.StatusForbidden, nil, &contracts.APIError{
			Code:    "FORBIDDEN",
			Message: "Hospital staff access is required",
		})
		return
	}
	patient, apiErr := resolveHospitalPatientLookup(r.Context(), claims.HospitalID, reqBody.PatientID)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}

	healthAddr := os.Getenv("HEALTH_RECORDS_SERVICE_GRPC_ADDR")
	if healthAddr == "" {
		healthAddr = "health-records-service:50054"
	}
	conn, err := grpcclient.DialInsecure(healthAddr)
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to connect to health records service: " + err.Error(),
		})
		return
	}
	defer conn.Close()

	ctx := metadata.AppendToOutgoingContext(
		r.Context(),
		"x-internal-token", os.Getenv("INTERNAL_SERVICE_SECRET"),
		"x-forwarded-token", accessToken,
	)
	client := healthpb.NewHealthRecordServiceClient(conn)
	resp, err := client.SummarizeRecords(ctx, &healthpb.SummarizeRequest{
		PatientId:  patient.PatientID,
		HospitalId: claims.HospitalID,
		Focus:      strings.TrimSpace(reqBody.Focus),
	})
	if err != nil {
		httpStatus := http.StatusInternalServerError
		apiCode := "INTERNAL_SERVER_ERROR"
		if s, ok := status.FromError(err); ok {
			switch s.Code() {
			case codes.InvalidArgument:
				httpStatus = http.StatusBadRequest
				apiCode = "BAD_REQUEST"
			case codes.NotFound:
				httpStatus = http.StatusNotFound
				apiCode = "NOT_FOUND"
			case codes.PermissionDenied:
				httpStatus = http.StatusForbidden
				apiCode = "FORBIDDEN"
			case codes.Unauthenticated:
				httpStatus = http.StatusUnauthorized
				apiCode = "UNAUTHORIZED"
			}
		}
		writeJson(w, httpStatus, nil, &contracts.APIError{
			Code:    apiCode,
			Message: "Failed to summarize patient records: " + err.Error(),
		})
		return
	}
	writeJson(w, http.StatusOK, HospitalPatientSummaryResponse{PatientID: patient.PatientID, Summary: resp.GetSummary()}, nil)
}

func PatientProfileHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}

	patientClient, conn, err := newPatientPortalClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to connect to patient service: " + err.Error(),
		})
		return
	}
	defer conn.Close()

	resp, err := patientClient.GetProfile(grpcclient.WithForwardedToken(r.Context(), accessToken), &patientportalpb.GetPatientProfileRequest{
		PatientId: claims.PatientID,
	})
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to load patient profile: " + err.Error(),
		})
		return
	}
	profile := resp.GetProfile()
	if profile == nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Patient service returned an empty profile",
		})
		return
	}

	writeJson(w, http.StatusOK, PatientProfileResponse{
		PatientID:     claims.PatientID,
		FullName:      profile.GetFullName(),
		Email:         profile.GetEmail(),
		PhoneNumber:   profile.GetPhoneNumber(),
		DateOfBirth:   profile.GetDateOfBirth(),
		MedicalNotes:  profile.GetMedicalNotes(),
		VoicePhone:    zorbaVoicePhoneNumber(),
		VoiceEnabled:  true,
		SupportWindow: env.GetString("ZORBA_SUPPORT_WINDOW", "24/7"),
	}, nil)
}

// zorbaVoicePhoneNumber is the PSTN number patients dial for the Zorba voice agent.
// Prefer an explicit display/override, then the VoIP.ms DID used by LiveKit SIP.
func zorbaVoicePhoneNumber() string {
	if v := strings.TrimSpace(env.GetString("ZORBA_VOICE_PHONE_NUMBER", "")); v != "" {
		return v
	}
	did := strings.TrimSpace(env.GetString("VOIPMS_DID", ""))
	if did == "" {
		return "+13185162690"
	}
	digits := make([]rune, 0, len(did))
	for _, r := range did {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	switch len(digits) {
	case 10:
		return "+1" + string(digits)
	case 11:
		if digits[0] == '1' {
			return "+" + string(digits)
		}
	}
	if strings.HasPrefix(did, "+") {
		return did
	}
	return "+" + string(digits)
}

func PatientListConsentsHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}

	auditClient, conn, err := newAuditPortalClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to connect to audit service: " + err.Error(),
		})
		return
	}
	defer conn.Close()

	resp, err := auditClient.ListPatientConsents(grpcclient.WithForwardedToken(r.Context(), accessToken), &auditportalpb.ListPatientConsentsRequest{
		PatientId:      claims.PatientID,
		IncludeRevoked: true,
		Limit:          100,
	})
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to load patient consents: " + err.Error(),
		})
		return
	}

	consents := make([]ConsentRecord, 0, len(resp.GetConsents()))
	for _, consent := range resp.GetConsents() {
		consents = append(consents, portalConsentFromProto(consent))
	}

	writeJson(w, http.StatusOK, PatientConsentListResponse{
		PatientID: claims.PatientID,
		Consents:  consents,
	}, nil)
}

func PatientGrantConsentHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}

	var reqBody PatientConsentMutationRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	defer r.Body.Close()

	if !isValidConsentType(reqBody.ConsentType) {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Unsupported consent_type",
		})
		return
	}

	auditClient, conn, err := newAuditClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to connect to audit service: " + err.Error(),
		})
		return
	}
	defer conn.Close()

	ctx := grpcclient.WithForwardedToken(r.Context(), accessToken)
	resp, err := auditClient.GrantConsent(ctx, &auditpb.GrantConsentRequest{
		PatientId:   claims.PatientID,
		ConsentType: reqBody.ConsentType,
		Scope:       strings.TrimSpace(reqBody.Scope),
		Source:      fallbackString(strings.TrimSpace(reqBody.Source), "patient-portal"),
		Metadata:    mapToStruct(reqBody.Metadata),
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "FORBIDDEN",
			Message: "Failed to grant consent: " + err.Error(),
		})
		return
	}

	writeJson(w, http.StatusOK, PatientConsentMutationResponse{
		Message: "Consent granted.",
		Consent: consentFromProto(resp.GetConsent()),
	}, nil)
}

func PatientRevokeConsentHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}

	var reqBody PatientConsentMutationRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	defer r.Body.Close()

	if !isValidConsentType(reqBody.ConsentType) {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Unsupported consent_type",
		})
		return
	}

	auditClient, conn, err := newAuditClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to connect to audit service: " + err.Error(),
		})
		return
	}
	defer conn.Close()

	ctx := grpcclient.WithForwardedToken(r.Context(), accessToken)
	resp, err := auditClient.RevokeConsent(ctx, &auditpb.RevokeConsentRequest{
		PatientId:   claims.PatientID,
		ConsentType: reqBody.ConsentType,
		Scope:       strings.TrimSpace(reqBody.Scope),
		Source:      fallbackString(strings.TrimSpace(reqBody.Source), "patient-portal"),
		Metadata:    mapToStruct(reqBody.Metadata),
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "FORBIDDEN",
			Message: "Failed to revoke consent: " + err.Error(),
		})
		return
	}

	writeJson(w, http.StatusOK, PatientConsentMutationResponse{
		Message: "Consent revoked.",
		Consent: consentFromProto(resp.GetConsent()),
	}, nil)
}

func PatientHealthAnswerHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}

	var reqBody PatientHealthAnswerRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}
	defer r.Body.Close()

	if strings.TrimSpace(reqBody.Question) == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "question is required",
		})
		return
	}

	healthClient, conn, err := newHealthRecordsClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to connect to health records service: " + err.Error(),
		})
		return
	}
	defer conn.Close()

	ctx := grpcclient.WithForwardedToken(r.Context(), accessToken)
	resp, err := healthClient.AnswerPatientQuestion(ctx, &healthpb.AnswerPatientQuestionRequest{
		PatientId: claims.PatientID,
		Question:  strings.TrimSpace(reqBody.Question),
		TopK:      defaultTopK(reqBody.TopK),
	})
	if err != nil {
		if s, ok := status.FromError(err); ok && s.Code() == codes.NotFound {
			writeJson(w, http.StatusNotFound, nil, &contracts.APIError{
				Code:    "NO_HEALTH_RECORDS",
				Message: "No health records are available yet. Once records are added, Zorba can answer questions with citations.",
			})
			return
		}
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "FORBIDDEN",
			Message: "Failed to answer health question: " + err.Error(),
		})
		return
	}

	citations := make([]PatientHealthCitation, 0, len(resp.GetCitations()))
	for _, item := range resp.GetCitations() {
		citations = append(citations, PatientHealthCitation{
			Text:       item.GetText(),
			SourceFile: item.GetSourceFile(),
			Score:      item.GetScore(),
		})
	}

	writeJson(w, http.StatusOK, PatientHealthAnswerResponse{
		Answer:    resp.GetAnswer(),
		Citations: citations,
	}, nil)
}

func PatientCallsHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}

	patientClient, conn, err := newPatientPortalClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to connect to patient service: " + err.Error(),
		})
		return
	}
	defer conn.Close()

	resp, err := patientClient.ListCallSummaries(grpcclient.WithForwardedToken(r.Context(), accessToken), &patientportalpb.ListPatientCallSummariesRequest{
		PatientId: claims.PatientID,
		Limit:     10,
	})
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to load call summaries: " + err.Error(),
		})
		return
	}

	calls := make([]PatientCallSummary, 0, len(resp.GetCalls()))
	for _, call := range resp.GetCalls() {
		calls = append(calls, PatientCallSummary{
			ID:            call.GetId(),
			Status:        call.GetStatus(),
			StartedAt:     protoTimeOrEmpty(call.GetStartedAt()),
			EndedAt:       protoTimeOrEmpty(call.GetEndedAt()),
			RecordingURL:  call.GetRecordingUrl(),
			Summary:       call.GetSummary(),
			LivekitRoomID: call.GetLivekitRoomId(),
		})
	}

	writeJson(w, http.StatusOK, PatientCallListResponse{
		PatientID: claims.PatientID,
		Calls:     calls,
	}, nil)
}

func PatientRequestBridgedCallTransferHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	var reqBody RequestBridgedCallTransferRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "Invalid request body: " + err.Error()})
		return
	}
	defer r.Body.Close()
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to connect to patient service: " + err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.RequestBridgedCallTransfer(grpcclient.WithForwardedToken(r.Context(), accessToken), &schedpb.RequestBridgedCallTransferRequest{
		SessionId:      strings.TrimSpace(reqBody.SessionID),
		RoomSid:        strings.TrimSpace(reqBody.RoomSID),
		PatientId:      claims.PatientID,
		HospitalId:     strings.TrimSpace(reqBody.HospitalID),
		StaffId:        strings.TrimSpace(reqBody.StaffID),
		TransferReason: strings.TrimSpace(reqBody.TransferReason),
	})
	if err != nil {
		writeSchedulingAPIError(w, "Failed to request bridged call transfer", err)
		return
	}
	writeJson(w, http.StatusOK, BridgedCallSessionResponse{
		Session:          bridgedCallSessionFromProto(resp.GetSession()),
		PatientRoomToken: resp.GetPatientRoomToken(),
		LiveKitWSURL:     resp.GetLivekitWsUrl(),
	}, nil)
}

func PatientGetBridgedCallSessionHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	handleGetBridgedCallSession(w, r, accessToken)
}

func PatientUpdateBridgedCallTranslationHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	handleUpdateBridgedCallTranslation(w, r, accessToken)
}

func PatientCreateWelfareCheckHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	var reqBody CreateWelfareCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "Invalid request body: " + err.Error()})
		return
	}
	defer r.Body.Close()
	scheduledAt, err := time.Parse(time.RFC3339, strings.TrimSpace(reqBody.ScheduledAt))
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "scheduled_at must be RFC3339"})
		return
	}
	client, conn, err := newPatientPortalClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to connect to patient service: " + err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.CreateWelfareCheck(grpcclient.WithForwardedToken(r.Context(), accessToken), &patientportalpb.CreateWelfareCheckRequest{
		PatientId: claims.PatientID, ScheduledAt: timestamppb.New(scheduledAt), Timezone: strings.TrimSpace(reqBody.Timezone),
		ReasonCode: strings.TrimSpace(reqBody.ReasonCode), ReasonDetail: strings.TrimSpace(reqBody.ReasonDetail),
	})
	if err != nil {
		writeGrpcAPIError(w, err, "WELFARE_CHECK_CREATE_FAILED")
		return
	}
	writeJson(w, http.StatusCreated, WelfareCheckResponse{WelfareCheck: welfareCheckFromProto(resp.GetWelfareCheck())}, nil)
}

func PatientListWelfareChecksHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	client, conn, err := newPatientPortalClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to connect to patient service: " + err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.ListWelfareChecks(grpcclient.WithForwardedToken(r.Context(), accessToken), &patientportalpb.ListWelfareChecksRequest{
		PatientId: claims.PatientID, IncludeCancelled: strings.EqualFold(r.URL.Query().Get("include_cancelled"), "true"), Limit: 50,
	})
	if err != nil {
		writeGrpcAPIError(w, err, "WELFARE_CHECK_LIST_FAILED")
		return
	}
	checks := make([]WelfareCheckRecord, 0, len(resp.GetWelfareChecks()))
	for _, check := range resp.GetWelfareChecks() {
		checks = append(checks, welfareCheckFromProto(check))
	}
	writeJson(w, http.StatusOK, WelfareCheckListResponse{WelfareChecks: checks}, nil)
}

func PatientCancelWelfareCheckHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	checkID := strings.TrimSpace(r.PathValue("id"))
	if checkID == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "welfare check id is required"})
		return
	}
	client, conn, err := newPatientPortalClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to connect to patient service: " + err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.CancelWelfareCheck(grpcclient.WithForwardedToken(r.Context(), accessToken), &patientportalpb.CancelWelfareCheckRequest{
		PatientId: claims.PatientID, WelfareCheckId: checkID,
	})
	if err != nil {
		writeGrpcAPIError(w, err, "WELFARE_CHECK_CANCEL_FAILED")
		return
	}
	writeJson(w, http.StatusOK, WelfareCheckResponse{WelfareCheck: welfareCheckFromProto(resp.GetWelfareCheck())}, nil)
}

func PatientAuditHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}

	auditClient, conn, err := newAuditPortalClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to connect to audit service: " + err.Error(),
		})
		return
	}
	defer conn.Close()

	resp, err := auditClient.ListPatientAuditEvents(grpcclient.WithForwardedToken(r.Context(), accessToken), &auditportalpb.ListPatientAuditEventsRequest{
		PatientId: claims.PatientID,
		Limit:     30,
	})
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to load patient audit trail: " + err.Error(),
		})
		return
	}

	events := portalAuditEventsFromProto(resp.GetEvents())

	writeJson(w, http.StatusOK, PatientAuditResponse{
		PatientID: claims.PatientID,
		Events:    events,
	}, nil)
}

func HospitalIncidentsHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}

	auditClient, conn, err := newAuditPortalClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to connect to audit service: " + err.Error(),
		})
		return
	}
	defer conn.Close()

	resp, err := auditClient.ListHospitalIncidents(grpcclient.WithForwardedToken(r.Context(), accessToken), &auditportalpb.ListHospitalIncidentsRequest{
		Limit: 30,
	})
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to load emergency incidents: " + err.Error(),
		})
		return
	}

	events := portalAuditEventsFromProto(resp.GetIncidents())

	incidents := make([]HospitalIncidentRecord, 0, len(events))
	for _, event := range events {
		if !incidentVisibleToHospital(event, claims.HospitalID) {
			continue
		}
		incidents = append(incidents, HospitalIncidentRecord{
			EventID:       event.EventID,
			PatientID:     event.PatientID,
			Timestamp:     event.Timestamp,
			Severity:      stringMetadata(event.Metadata, "severity"),
			SessionID:     stringMetadata(event.Metadata, "session_id"),
			ServiceName:   event.ServiceName,
			FailureReason: event.FailureReason,
			Metadata:      scrubIncidentMetadata(event.Metadata),
		})
	}

	writeJson(w, http.StatusOK, HospitalIncidentListResponse{Incidents: incidents}, nil)
}

func HospitalPatientAuditHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}

	patientID := strings.TrimSpace(r.URL.Query().Get("patient_id"))
	if patientID == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "patient_id, name, or email query parameter is required",
		})
		return
	}
	patient, apiErr := resolveHospitalPatientLookup(r.Context(), claims.HospitalID, patientID)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	auditClient, conn, err := newAuditPortalClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: "Failed to connect to audit service: " + err.Error(),
		})
		return
	}
	defer conn.Close()

	resp, err := auditClient.ListHospitalPatientAuditEvents(grpcclient.WithForwardedToken(r.Context(), accessToken), &auditportalpb.ListHospitalPatientAuditEventsRequest{
		PatientId: patient.PatientID,
		Limit:     30,
	})
	if err != nil {
		code := http.StatusInternalServerError
		if s, ok := status.FromError(err); ok && s.Code() == codes.PermissionDenied {
			code = http.StatusForbidden
		}
		writeJson(w, code, nil, &contracts.APIError{
			Code:    "FORBIDDEN",
			Message: "Failed to load patient audit trail: " + err.Error(),
		})
		return
	}

	events := portalAuditEventsFromProto(resp.GetEvents())

	writeJson(w, http.StatusOK, HospitalPatientAuditResponse{
		PatientID: patient.PatientID,
		Events:    events,
	}, nil)
}

func HospitalConnectBridgedCallHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	var reqBody ConnectBridgedCallRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "Invalid request body: " + err.Error()})
		return
	}
	defer r.Body.Close()
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to connect to patient service: " + err.Error()})
		return
	}
	defer conn.Close()
	joinMode := strings.ToLower(strings.TrimSpace(reqBody.JoinMode))
	if joinMode == "" {
		joinMode = "web"
	}
	ctx := grpcclient.WithForwardedToken(r.Context(), accessToken)
	ctx = metadata.AppendToOutgoingContext(ctx, "x-bridge-join-mode", joinMode)
	resp, err := client.ConnectBridgedCall(ctx, &schedpb.ConnectBridgedCallRequest{
		SessionId:                strings.TrimSpace(reqBody.SessionID),
		StaffId:                  claims.StaffID,
		StaffParticipantIdentity: strings.TrimSpace(reqBody.StaffParticipantIdentity),
	})
	if err != nil {
		writeSchedulingAPIError(w, "Failed to connect bridged call", err)
		return
	}
	writeJson(w, http.StatusOK, BridgedCallConnectResponse{
		Session:        bridgedCallSessionFromProto(resp.GetSession()),
		StaffRoomToken: resp.GetStaffRoomToken(),
		LiveKitWSURL:   resp.GetLivekitWsUrl(),
	}, nil)
}

func HospitalListBridgedCallSessionsHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to connect to patient service: " + err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.ListBridgedCallSessions(grpcclient.WithForwardedToken(r.Context(), accessToken), &schedpb.ListBridgedCallSessionsRequest{
		Status: strings.TrimSpace(r.URL.Query().Get("status")),
		Limit:  50,
	})
	if err != nil {
		writeSchedulingAPIError(w, "Failed to list bridged call sessions", err)
		return
	}
	sessions := make([]BridgedCallSessionRecord, 0, len(resp.GetSessions()))
	for _, session := range resp.GetSessions() {
		sessions = append(sessions, bridgedCallSessionFromProto(session))
	}
	writeJson(w, http.StatusOK, BridgedCallSessionListResponse{Sessions: sessions}, nil)
}

func HospitalGetBridgedCallSessionHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	handleGetBridgedCallSession(w, r, accessToken)
}

func HospitalUpdateBridgedCallTranslationHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	handleUpdateBridgedCallTranslation(w, r, accessToken)
}

func HospitalEndBridgedCallHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	handleEndBridgedCall(w, r, accessToken)
}

func PatientEndBridgedCallHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, _, apiErr := requirePatientClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	handleEndBridgedCall(w, r, accessToken)
}

func handleEndBridgedCall(w http.ResponseWriter, r *http.Request, accessToken string) {
	var reqBody EndBridgedCallRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "Invalid request body: " + err.Error()})
		return
	}
	defer r.Body.Close()
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to connect to patient service: " + err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.EndBridgedCall(grpcclient.WithForwardedToken(r.Context(), accessToken), &schedpb.EndBridgedCallRequest{
		SessionId: strings.TrimSpace(reqBody.SessionID),
		Reason:    strings.TrimSpace(reqBody.Reason),
	})
	if err != nil {
		writeSchedulingAPIError(w, "Failed to end bridged call", err)
		return
	}
	writeJson(w, http.StatusOK, BridgedCallSessionResponse{
		Session: bridgedCallSessionFromProto(resp.GetSession()),
	}, nil)
}

func requirePatientClaims(r *http.Request) (string, *sharedauth.Claims, *contracts.APIError) {
	accessToken := bearerToken(r)
	if accessToken == "" {
		return "", nil, &contracts.APIError{Code: "UNAUTHORIZED", Message: "Authorization bearer token is required"}
	}
	claims, err := sharedauth.VerifyToken(accessToken)
	if err != nil {
		return "", nil, &contracts.APIError{Code: "UNAUTHORIZED", Message: "Invalid patient token"}
	}
	if claims.ActorType != sharedauth.ActorPatient || strings.TrimSpace(claims.PatientID) == "" {
		return "", nil, &contracts.APIError{Code: "FORBIDDEN", Message: "Patient access is required"}
	}
	return accessToken, claims, nil
}

func requireStaffClaims(r *http.Request) (string, *sharedauth.Claims, *contracts.APIError) {
	accessToken := bearerToken(r)
	if accessToken == "" {
		return "", nil, &contracts.APIError{Code: "UNAUTHORIZED", Message: "Authorization bearer token is required"}
	}
	claims, err := sharedauth.VerifyToken(accessToken)
	if err != nil {
		return "", nil, &contracts.APIError{Code: "UNAUTHORIZED", Message: "Invalid hospital token"}
	}
	if claims.ActorType != sharedauth.ActorStaff || strings.TrimSpace(claims.HospitalID) == "" {
		return "", nil, &contracts.APIError{Code: "FORBIDDEN", Message: "Hospital staff access is required"}
	}
	return accessToken, claims, nil
}

func bearerToken(r *http.Request) string {
	return strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
}

func patientLoginIdentifiers(req PatientLoginRequest) (email string, phoneNumber string) {
	email = strings.TrimSpace(strings.ToLower(req.Email))
	phoneNumber = strings.TrimSpace(req.PhoneNumber)

	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		return email, phoneNumber
	}
	if strings.Contains(identifier, "@") {
		if email == "" {
			email = strings.ToLower(identifier)
		}
		return email, phoneNumber
	}
	if phoneNumber == "" {
		phoneNumber = identifier
	}
	return email, phoneNumber
}

func statusCodeForAPIError(err *contracts.APIError) int {
	if err == nil {
		return http.StatusInternalServerError
	}
	switch err.Code {
	case "INVALID_REQUEST_BODY":
		return http.StatusBadRequest
	case "UNAUTHORIZED":
		return http.StatusUnauthorized
	case "FORBIDDEN":
		return http.StatusForbidden
	case "NOT_FOUND":
		return http.StatusNotFound
	case "AMBIGUOUS_PATIENT_LOOKUP":
		return http.StatusConflict
	case "SERVICE_UNAVAILABLE":
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func writeGrpcAPIError(w http.ResponseWriter, err error, code string) {
	httpStatus := http.StatusInternalServerError
	message := err.Error()
	if st, ok := status.FromError(err); ok {
		message = st.Message()
		switch st.Code() {
		case codes.InvalidArgument:
			httpStatus = http.StatusBadRequest
		case codes.NotFound:
			httpStatus = http.StatusNotFound
		case codes.FailedPrecondition:
			httpStatus = http.StatusPreconditionFailed
		case codes.PermissionDenied:
			httpStatus = http.StatusForbidden
		case codes.Unauthenticated:
			httpStatus = http.StatusUnauthorized
		default:
			httpStatus = http.StatusInternalServerError
		}
	}
	writeJson(w, httpStatus, nil, &contracts.APIError{Code: code, Message: message})
}

func newHealthRecordsClient() (healthpb.HealthRecordServiceClient, *grpc.ClientConn, error) {
	healthAddr := os.Getenv("HEALTH_RECORDS_SERVICE_GRPC_ADDR")
	if healthAddr == "" {
		healthAddr = "health-records-service:50054"
	}
	conn, err := grpcclient.Dial(healthAddr)
	if err != nil {
		return nil, nil, err
	}
	return healthpb.NewHealthRecordServiceClient(conn), conn, nil
}

func newAuditClient() (auditpb.AuditServiceClient, *grpc.ClientConn, error) {
	auditAddr := os.Getenv("AUDIT_SERVICE_GRPC_ADDR")
	if auditAddr == "" {
		auditAddr = "audit-service:50058"
	}
	conn, err := grpcclient.Dial(auditAddr)
	if err != nil {
		return nil, nil, err
	}
	return auditpb.NewAuditServiceClient(conn), conn, nil
}

func newAuditPortalClient() (auditportalpb.AuditPortalServiceClient, *grpc.ClientConn, error) {
	auditAddr := os.Getenv("AUDIT_SERVICE_GRPC_ADDR")
	if auditAddr == "" {
		auditAddr = "audit-service:50058"
	}
	conn, err := grpcclient.Dial(auditAddr)
	if err != nil {
		return nil, nil, err
	}
	return auditportalpb.NewAuditPortalServiceClient(conn), conn, nil
}

func newPatientPortalClient() (patientportalpb.PatientPortalServiceClient, *grpc.ClientConn, error) {
	patientAddr := os.Getenv("PATIENT_SERVICE_GRPC_ADDR")
	if patientAddr == "" {
		patientAddr = "patient-service:9093"
	}
	conn, err := grpcclient.Dial(patientAddr)
	if err != nil {
		return nil, nil, err
	}
	return patientportalpb.NewPatientPortalServiceClient(conn), conn, nil
}

func newSchedulingClientFromEnv() (schedpb.SchedulingServiceClient, *grpc.ClientConn, error) {
	patientAddr := os.Getenv("PATIENT_SERVICE_GRPC_ADDR")
	if patientAddr == "" {
		patientAddr = "patient-service:9093"
	}
	conn, err := grpcclient.Dial(patientAddr)
	if err != nil {
		return nil, nil, err
	}
	return schedpb.NewSchedulingServiceClient(conn), conn, nil
}

func incidentVisibleToHospital(event AuditEventRecord, hospitalID string) bool {
	if hospitalID == "" {
		return false
	}
	target := stringMetadata(event.Metadata, "hospital_id")
	return target == "" || target == hospitalID
}

func scrubIncidentMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return map[string]any{}
	}
	out := map[string]any{}
	for _, key := range []string{"severity", "session_id", "transfer_requested", "transfer_target"} {
		if value, ok := metadata[key]; ok {
			out[key] = value
		}
	}
	return out
}

func isValidConsentType(value string) bool {
	for _, consentType := range sharedaudit.AllConsentTypes {
		if value == consentType {
			return true
		}
	}
	return false
}

func defaultTopK(value int32) int32 {
	if value <= 0 {
		return 5
	}
	return value
}

func fallbackString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func handleGetBridgedCallSession(w http.ResponseWriter, r *http.Request, accessToken string) {
	sessionID := strings.TrimSpace(r.URL.Query().Get("session_id"))
	if sessionID == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "session_id query parameter is required"})
		return
	}
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to connect to patient service: " + err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.GetBridgedCallSession(grpcclient.WithForwardedToken(r.Context(), accessToken), &schedpb.GetBridgedCallSessionRequest{SessionId: sessionID})
	if err != nil {
		writeSchedulingAPIError(w, "Failed to load bridged call session", err)
		return
	}
	writeJson(w, http.StatusOK, BridgedCallSessionResponse{
		Session:          bridgedCallSessionFromProto(resp.GetSession()),
		PatientRoomToken: resp.GetPatientRoomToken(),
		LiveKitWSURL:     resp.GetLivekitWsUrl(),
	}, nil)
}

func handleUpdateBridgedCallTranslation(w http.ResponseWriter, r *http.Request, accessToken string) {
	var reqBody UpdateBridgedCallTranslationRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "Invalid request body: " + err.Error()})
		return
	}
	defer r.Body.Close()
	client, conn, err := newSchedulingClientFromEnv()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: "Failed to connect to patient service: " + err.Error()})
		return
	}
	defer conn.Close()
	resp, err := client.UpdateBridgedCallTranslation(grpcclient.WithForwardedToken(r.Context(), accessToken), &schedpb.UpdateBridgedCallTranslationRequest{
		SessionId:   strings.TrimSpace(reqBody.SessionID),
		Participant: strings.TrimSpace(reqBody.Participant),
		Translation: bridgedCallTranslationToProto(reqBody.Translation),
	})
	if err != nil {
		writeSchedulingAPIError(w, "Failed to update bridged call translation", err)
		return
	}
	writeJson(w, http.StatusOK, BridgedCallSessionResponse{Session: bridgedCallSessionFromProto(resp.GetSession())}, nil)
}

func writeSchedulingAPIError(w http.ResponseWriter, prefix string, err error) {
	code := http.StatusInternalServerError
	apiCode := "INTERNAL_SERVER_ERROR"
	if s, ok := status.FromError(err); ok {
		switch s.Code() {
		case codes.NotFound:
			code = http.StatusNotFound
			apiCode = "NOT_FOUND"
		case codes.PermissionDenied:
			code = http.StatusForbidden
			apiCode = "FORBIDDEN"
		case codes.InvalidArgument:
			code = http.StatusBadRequest
			apiCode = "INVALID_REQUEST_BODY"
		case codes.FailedPrecondition:
			code = http.StatusConflict
			apiCode = "FAILED_PRECONDITION"
		case codes.Unavailable:
			code = http.StatusServiceUnavailable
			apiCode = "SERVICE_UNAVAILABLE"
		case codes.Unauthenticated:
			code = http.StatusUnauthorized
			apiCode = "UNAUTHORIZED"
		}
	}
	writeJson(w, code, nil, &contracts.APIError{Code: apiCode, Message: prefix + ": " + err.Error()})
}

func bridgedCallSessionFromProto(session *schedpb.BridgedCallSession) BridgedCallSessionRecord {
	if session == nil {
		return BridgedCallSessionRecord{}
	}
	return BridgedCallSessionRecord{
		SessionID:            session.GetSessionId(),
		RoomSID:              session.GetRoomSid(),
		PatientID:            session.GetPatientId(),
		HospitalID:           session.GetHospitalId(),
		StaffID:              session.GetStaffId(),
		Status:               session.GetStatus(),
		RequestedByActorType: session.GetRequestedByActorType(),
		RequestedByActorID:   session.GetRequestedByActorId(),
		TransferReason:       session.GetTransferReason(),
		RequestedAt:          protoTimeOrEmpty(session.GetRequestedAt()),
		ConnectedAt:          protoTimeOrEmpty(session.GetConnectedAt()),
		EndedAt:              protoTimeOrEmpty(session.GetEndedAt()),
		PatientTranslation:   bridgedCallTranslationFromProto(session.GetPatientTranslation()),
		StaffTranslation:     bridgedCallTranslationFromProto(session.GetStaffTranslation()),
	}
}

func bridgedCallTranslationFromProto(p *schedpb.BridgedCallTranslationPreferences) BridgedCallTranslationPreferencesRecord {
	if p == nil {
		return BridgedCallTranslationPreferencesRecord{}
	}
	return BridgedCallTranslationPreferencesRecord{
		Enabled:             p.GetEnabled(),
		LanguageMode:        p.GetLanguageMode(),
		LanguageCode:        p.GetLanguageCode(),
		ParticipantIdentity: p.GetParticipantIdentity(),
		UpdatedAt:           protoTimeOrEmpty(p.GetUpdatedAt()),
	}
}

func bridgedCallTranslationToProto(p BridgedCallTranslationPreferencesRecord) *schedpb.BridgedCallTranslationPreferences {
	return &schedpb.BridgedCallTranslationPreferences{
		Enabled:             p.Enabled,
		LanguageMode:        strings.TrimSpace(p.LanguageMode),
		LanguageCode:        strings.TrimSpace(strings.ToLower(p.LanguageCode)),
		ParticipantIdentity: strings.TrimSpace(p.ParticipantIdentity),
	}
}

func consentFromProto(consent *auditpb.Consent) ConsentRecord {
	if consent == nil {
		return ConsentRecord{}
	}
	return ConsentRecord{
		ConsentID:      consent.GetConsentId(),
		ConsentType:    consent.GetConsentType(),
		GrantedBy:      consent.GetGrantedBy(),
		GrantedAt:      protoTimeOrEmpty(consent.GetGrantedAt()),
		RevokedAt:      protoTimeOrEmpty(consent.GetRevokedAt()),
		Scope:          consent.GetScope(),
		ExpirationTime: protoTimeOrEmpty(consent.GetExpirationTime()),
		Source:         consent.GetSource(),
		Status:         protoConsentStatus(consent),
		Metadata:       consent.GetMetadata().AsMap(),
	}
}

func portalConsentFromProto(consent *auditportalpb.PortalConsent) ConsentRecord {
	if consent == nil {
		return ConsentRecord{}
	}
	metadata := map[string]any{}
	if consent.GetMetadata() != nil {
		metadata = consent.GetMetadata().AsMap()
	}
	return ConsentRecord{
		ConsentID:      consent.GetConsentId(),
		ConsentType:    consent.GetConsentType(),
		GrantedBy:      consent.GetGrantedBy(),
		GrantedAt:      protoTimeOrEmpty(consent.GetGrantedAt()),
		RevokedAt:      protoTimeOrEmpty(consent.GetRevokedAt()),
		Scope:          consent.GetScope(),
		ExpirationTime: protoTimeOrEmpty(consent.GetExpirationTime()),
		Source:         consent.GetSource(),
		Status:         portalConsentStatus(consent),
		Metadata:       metadata,
	}
}

func portalConsentStatus(consent *auditportalpb.PortalConsent) string {
	if consent == nil {
		return "unknown"
	}
	if consent.GetRevokedAt() != nil {
		return "revoked"
	}
	if consent.GetExpirationTime() != nil && consent.GetExpirationTime().AsTime().Before(time.Now()) {
		return "expired"
	}
	return "active"
}

func portalAuditEventsFromProto(events []*auditportalpb.PortalAuditEvent) []AuditEventRecord {
	out := make([]AuditEventRecord, 0, len(events))
	for _, event := range events {
		metadata := map[string]any{}
		if event.GetMetadata() != nil {
			metadata = event.GetMetadata().AsMap()
		}
		out = append(out, AuditEventRecord{
			EventID:       event.GetEventId(),
			EventType:     event.GetEventType(),
			ActorType:     event.GetActorType(),
			ActorID:       event.GetActorId(),
			PatientID:     event.GetPatientId(),
			ServiceName:   event.GetServiceName(),
			ResourceType:  event.GetResourceType(),
			ResourceID:    event.GetResourceId(),
			Timestamp:     protoTimeOrEmpty(event.GetTimestamp()),
			CorrelationID: event.GetCorrelationId(),
			ToolName:      event.GetToolName(),
			SuccessStatus: event.GetSuccessStatus(),
			FailureReason: event.GetFailureReason(),
			Metadata:      metadata,
		})
	}
	return out
}

func protoConsentStatus(consent *auditpb.Consent) string {
	if consent == nil {
		return "unknown"
	}
	if consent.GetRevokedAt() != nil {
		return "revoked"
	}
	if consent.GetExpirationTime() != nil && consent.GetExpirationTime().AsTime().Before(time.Now()) {
		return "expired"
	}
	return "active"
}

func mapToStruct(value map[string]any) *structpb.Struct {
	if len(value) == 0 {
		return &structpb.Struct{}
	}
	s, err := structpb.NewStruct(value)
	if err != nil {
		return &structpb.Struct{}
	}
	return s
}

func stringMetadata(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	value, ok := metadata[key]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func protoTimeOrEmpty(value *timestamppb.Timestamp) string {
	if value == nil {
		return ""
	}
	t := value.AsTime()
	if t.IsZero() || t.Unix() <= 0 {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func welfareCheckFromProto(check *patientportalpb.WelfareCheck) WelfareCheckRecord {
	if check == nil {
		return WelfareCheckRecord{}
	}
	return WelfareCheckRecord{
		ID:                     check.GetId(),
		PatientID:              check.GetPatientId(),
		ScheduledAt:            protoTimeOrEmpty(check.GetScheduledAt()),
		Timezone:               check.GetTimezone(),
		ReasonCode:             check.GetReasonCode(),
		ReasonDetail:           check.GetReasonDetail(),
		Status:                 check.GetStatus(),
		RecurrenceRule:         check.GetRecurrenceRule(),
		CreatedAt:              protoTimeOrEmpty(check.GetCreatedAt()),
		UpdatedAt:              protoTimeOrEmpty(check.GetUpdatedAt()),
		CancelledAt:            protoTimeOrEmpty(check.GetCancelledAt()),
		LatestRunID:            check.GetLatestRunId(),
		LatestRunStatus:        check.GetLatestRunStatus(),
		LatestRunAttempts:      check.GetLatestRunAttempts(),
		LatestRunFailureReason: check.GetLatestRunFailureReason(),
	}
}
