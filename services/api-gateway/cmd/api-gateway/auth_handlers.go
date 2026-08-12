package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/services/api-gateway/cmd/api-gateway/grpc_clients"
	sharedauth "github.com/KoiralaSam/ZorbaHealth/shared/auth"
	"github.com/KoiralaSam/ZorbaHealth/shared/contracts"
	sharedlogging "github.com/KoiralaSam/ZorbaHealth/shared/logging"
	authpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/auth"
	healthproviderpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/health_provider"
)

type refreshBody struct {
	RefreshToken string `json:"refresh_token"`
}

func PatientRefreshHandler(w http.ResponseWriter, r *http.Request) {
	handleRefresh(w, r, "patient", cookieNamePatientRefresh, cookiePathPatientRefresh)
}

func PatientLogoutHandler(w http.ResponseWriter, r *http.Request) {
	handleLogout(w, r, cookieNamePatientRefresh, cookiePathPatientRefresh)
}

func HospitalRefreshHandler(w http.ResponseWriter, r *http.Request) {
	handleRefresh(w, r, "staff", cookieNameHospitalRefresh, cookiePathHospitalRefresh)
}

func HospitalLogoutHandler(w http.ResponseWriter, r *http.Request) {
	handleLogout(w, r, cookieNameHospitalRefresh, cookiePathHospitalRefresh)
}

func HospitalRegisterHandler(w http.ResponseWriter, r *http.Request) {
	if !allowLoginRate(r, "hospital-register", clientIP(r)) {
		writeRateLimited(w)
		return
	}
	var body HospitalRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "Invalid request body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	hospitalName := strings.TrimSpace(body.HospitalName)
	licenseNo := strings.TrimSpace(body.LicenseNo)
	if licenseNo == "" {
		licenseNo = strings.TrimSpace(body.RegistrationNumber)
	}
	staffName := strings.TrimSpace(body.StaffName)
	email := strings.TrimSpace(strings.ToLower(body.Email))
	if hospitalName == "" || licenseNo == "" || staffName == "" || email == "" || strings.TrimSpace(body.Password) == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "hospital_name, license_no, staff_name, email, and password are required",
		})
		return
	}

	client, err := grpc_clients.NewHealthProviderServiceClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer client.Close()

	resp, err := client.RegisterClient.RegisterHospital(r.Context(), &healthproviderpb.RegisterHospitalRequest{
		HospitalName: hospitalName,
		LicenseNo:    licenseNo,
		StaffName:    staffName,
		Email:        email,
		PhoneNumber:  strings.TrimSpace(body.PhoneNumber),
		Password:     body.Password,
		StaffRole:    "admin",
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "REGISTRATION_FAILED", Message: "Hospital registration failed: " + err.Error()})
		return
	}

	writeJson(w, http.StatusCreated, HospitalRegisterResponse{
		Message:    resp.GetMessage(),
		UserID:     resp.GetUserId(),
		HospitalID: resp.GetHospitalId(),
		StaffID:    resp.GetStaffId(),
		StaffRole:  resp.GetStaffRole(),
	}, nil)
}

func HospitalStaffRegisterHandler(w http.ResponseWriter, r *http.Request) {
	_, claims, apiErr := requireStaffClaims(r)
	if apiErr != nil {
		writeJson(w, statusCodeForAPIError(apiErr), nil, apiErr)
		return
	}
	if claims.Role != "admin" {
		writeJson(w, http.StatusForbidden, nil, &contracts.APIError{Code: "FORBIDDEN", Message: "Only hospital admins can register staff"})
		return
	}

	var body HospitalStaffRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "INVALID_REQUEST_BODY", Message: "Invalid request body: " + err.Error()})
		return
	}
	defer r.Body.Close()

	email := strings.TrimSpace(strings.ToLower(body.Email))
	staffName := strings.TrimSpace(body.StaffName)
	staffRole := strings.TrimSpace(strings.ToLower(body.StaffRole))
	if staffRole == "" {
		staffRole = "doctor"
	}
	if email == "" || staffName == "" || strings.TrimSpace(body.Password) == "" {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{
			Code:    "INVALID_REQUEST_BODY",
			Message: "email, staff_name, and password are required",
		})
		return
	}

	client, err := grpc_clients.NewHealthProviderServiceClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer client.Close()

	resp, err := client.RegisterClient.RegisterHospitalStaff(r.Context(), &healthproviderpb.RegisterHospitalStaffRequest{
		HospitalId:  claims.HospitalID,
		StaffName:   staffName,
		Email:       email,
		PhoneNumber: strings.TrimSpace(body.PhoneNumber),
		Password:    body.Password,
		StaffRole:   staffRole,
	})
	if err != nil {
		writeJson(w, http.StatusBadRequest, nil, &contracts.APIError{Code: "REGISTRATION_FAILED", Message: "Staff registration failed: " + err.Error()})
		return
	}

	writeJson(w, http.StatusCreated, HospitalRegisterResponse{
		Message:    resp.GetMessage(),
		UserID:     resp.GetUserId(),
		HospitalID: resp.GetHospitalId(),
		StaffID:    resp.GetStaffId(),
		StaffRole:  resp.GetStaffRole(),
	}, nil)
}

func handleRefresh(w http.ResponseWriter, r *http.Request, actorType, cookieName, cookiePath string) {
	if !allowRefreshRate(r) {
		writeRateLimited(w)
		return
	}
	var body refreshBody
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}
	token := refreshTokenFromRequest(r, cookieName, body.RefreshToken)
	if token == "" {
		writeJson(w, http.StatusUnauthorized, nil, &contracts.APIError{Code: "UNAUTHORIZED", Message: "refresh token required"})
		return
	}
	client, err := grpc_clients.NewPatientAuthServiceClient()
	if err != nil {
		writeJson(w, http.StatusInternalServerError, nil, &contracts.APIError{Code: "INTERNAL_SERVER_ERROR", Message: err.Error()})
		return
	}
	defer client.Close()

	resp, err := client.RefreshClient.RefreshSession(r.Context(), &authpb.RefreshSessionRequest{
		RefreshToken:      token,
		ExpectedActorType: actorType,
		ClientKind:        clientKindFromRequest(r),
	})
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "REFRESH_TOKEN_REUSE") {
			clearRefreshCookie(w, cookieName, cookiePath)
			writeJson(w, http.StatusUnauthorized, nil, &contracts.APIError{Code: "REFRESH_TOKEN_REUSE", Message: "Session expired for your security. Please sign in again."})
			return
		}
		writeJson(w, http.StatusUnauthorized, nil, &contracts.APIError{Code: "UNAUTHORIZED", Message: "Invalid or expired session"})
		return
	}
	setRefreshCookie(w, cookieName, resp.RefreshToken, cookiePath)
	payload := map[string]any{
		"access_token": resp.AccessToken,
		"user_id":      resp.UserId,
		"auth_uuid":    resp.AuthUuid,
		"role":         resp.Role,
	}
	if actorType == sharedauth.ActorPatient {
		if claims, err := sharedauth.VerifyToken(resp.AccessToken); err == nil {
			payload["patient_id"] = claims.PatientID
		}
	}
	if actorType == sharedauth.ActorStaff {
		if claims, err := sharedauth.VerifyToken(resp.AccessToken); err == nil {
			payload["hospital_id"] = claims.HospitalID
			payload["staff_id"] = claims.StaffID
			if claims.Role != "" {
				payload["role"] = claims.Role
			}
		}
	}
	if clientKindFromRequest(r) == "mobile" {
		payload["refresh_token"] = resp.RefreshToken
	}
	writeJson(w, http.StatusOK, payload, nil)
}

func handleLogout(w http.ResponseWriter, r *http.Request, cookieName, cookiePath string) {
	access := authBearerToken(r)
	client, err := grpc_clients.NewPatientAuthServiceClient()
	if err == nil {
		defer client.Close()
		if access != "" {
			_, _ = client.LogoutRefreshClient.Logout(r.Context(), &authpb.LogoutRequest{AccessToken: access})
		}
	}
	clearRefreshCookie(w, cookieName, cookiePath)
	writeJson(w, http.StatusOK, map[string]string{"message": "logged out"}, nil)
}

func authBearerToken(r *http.Request) string {
	return strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
}

func loginRateLimitKeys(r *http.Request, actor, identifier string) (idKey, ipKey string) {
	idKey = "login:identifier:" + actor + ":" + sharedlogging.HashIdentifier(identifier)
	ipKey = "login:ip:" + sharedlogging.HashIdentifier(clientIP(r))
	return idKey, ipKey
}

func allowLoginRate(r *http.Request, actor, identifier string) bool {
	idKey, ipKey := loginRateLimitKeys(r, actor, identifier)
	ok1, _, _ := gatewayRateLimiter.Allow(r.Context(), idKey, envInt("AUTH_LOGIN_LIMIT_PER_IDENTIFIER", 10), 15*time.Minute)
	ok2, _, _ := gatewayRateLimiter.Allow(r.Context(), ipKey, envInt("AUTH_LOGIN_LIMIT_PER_IP", 30), 15*time.Minute)
	return ok1 && ok2
}

func allowRefreshRate(r *http.Request) bool {
	ipKey := "refresh:ip:" + sharedlogging.HashIdentifier(clientIP(r))
	ok, _, _ := gatewayRateLimiter.Allow(r.Context(), ipKey, envInt("AUTH_REFRESH_LIMIT_PER_IP", 60), time.Minute)
	return ok
}

func writeRateLimited(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "60")
	writeJson(w, http.StatusTooManyRequests, nil, &contracts.APIError{
		Code:    "RATE_LIMITED",
		Message: "Too many attempts. Try again later.",
	})
}
