package main

import (
	"context"
	"log"
	"net/http"

	"github.com/KoiralaSam/ZorbaHealth/shared/db"
	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/KoiralaSam/ZorbaHealth/shared/tracing"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var (
	httpAddr = env.GetString("API_GATEWAY_HTTP_ADDR", ":8081")
)

func main() {
	log.Println("Starting API Gateway")
	tracerCfg := tracing.Config{
		ServiceName:    "api-gateway",
		Environment:    env.GetString("ENVIRONMENT", "development"),
		JaegerEndpoint: env.GetString("JAEGER_ENDPOINT", "http://localhost:4318/v1/traces"),
	}
	shutdown, err := tracing.InitTracer(tracerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := shutdown(context.Background()); err != nil {
			log.Printf("tracer shutdown: %v", err)
		}
	}()

	if dbURL := env.GetString("DATABASE_URL", ""); dbURL != "" {
		if err := db.InitDB(context.Background(), dbURL); err != nil {
			log.Printf("database unavailable; DB-backed routes disabled: %v", err)
		}
	}
	if db.GetDB() != nil {
		defer db.GetDB().Close()
	}

	mux := http.NewServeMux()

	// CORS preflight: OPTIONS must be handled for each path (browser sends OPTIONS before POST)
	optCORS := corsMiddleware(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/login", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/register", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/register/verify", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/register/verify-otp", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/hospital/login", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/hospital/register", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/hospital/staff/register", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/refresh", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/patient/logout", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/hospital/refresh", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/auth/hospital/logout", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/patients", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/records/summary", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/profile", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/consents", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/hospital-consents", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/hospital-consents/{hospital_id}", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/consent-requests/{token}", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/consent-requests/{token}/approve", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/records/answer", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/calls", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/welfare-checks", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/welfare-checks/{id}", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/calls/bridge-transfer", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/calls/bridge-session", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/calls/bridge-translation", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/calls/bridge-end", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/audit", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/incidents", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/patient/audit", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/calls/bridge-connect", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/calls/bridge-session", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/calls/bridge-translation", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/calls/bridge-end", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/calls/bridge-sessions", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/meetings", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/meetings/{id}", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/appointments", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/appointments/{id}", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/appointments/{id}/reschedule", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/appointment-slots", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/appointments", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/appointments/{id}", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/appointment-slots", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/availability", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/availability/exceptions", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/availability/exceptions/{id}", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/patient/schedulable-staff", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/meetings", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/meetings/{id}", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/meetings/{id}/accept", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/meetings/{id}/reschedule", optCORS)
	mux.HandleFunc("OPTIONS /api/v1/hospital/consent-requests", optCORS)

	// API routes with CORS (Go 1.22+ requires space between method and path)
	mux.Handle("POST /api/v1/auth/patient/login", tracing.WrapHandlerFunc(corsMiddleware(PatientLoginHandler), "/api/v1/auth/patient/login"))
	mux.Handle("POST /api/v1/auth/patient/register", tracing.WrapHandlerFunc(corsMiddleware(PatientRegisterHandler), "/api/v1/auth/patient/register"))
	mux.Handle("POST /api/v1/auth/patient/register/verify", tracing.WrapHandlerFunc(corsMiddleware(PatientRegisterVerifyHandler), "/api/v1/auth/patient/register/verify"))
	mux.Handle("POST /api/v1/auth/patient/register/verify-otp", tracing.WrapHandlerFunc(corsMiddleware(PatientRegisterVerifyOTPHandler), "/api/v1/auth/patient/register/verify-otp"))
	mux.Handle("POST /api/v1/auth/hospital/login", tracing.WrapHandlerFunc(corsMiddleware(HospitalLoginHandler), "/api/v1/auth/hospital/login"))
	mux.Handle("POST /api/v1/auth/hospital/register", tracing.WrapHandlerFunc(corsMiddleware(HospitalRegisterHandler), "/api/v1/auth/hospital/register"))
	mux.Handle("POST /api/v1/auth/hospital/staff/register", tracing.WrapHandlerFunc(corsMiddleware(HospitalStaffRegisterHandler), "/api/v1/auth/hospital/staff/register"))
	mux.Handle("POST /api/v1/auth/patient/refresh", tracing.WrapHandlerFunc(corsMiddleware(PatientRefreshHandler), "/api/v1/auth/patient/refresh"))
	mux.Handle("POST /api/v1/auth/patient/logout", tracing.WrapHandlerFunc(corsMiddleware(PatientLogoutHandler), "/api/v1/auth/patient/logout"))
	mux.Handle("POST /api/v1/auth/hospital/refresh", tracing.WrapHandlerFunc(corsMiddleware(HospitalRefreshHandler), "/api/v1/auth/hospital/refresh"))
	mux.Handle("POST /api/v1/auth/hospital/logout", tracing.WrapHandlerFunc(corsMiddleware(HospitalLogoutHandler), "/api/v1/auth/hospital/logout"))
	mux.Handle("GET /api/v1/hospital/patients", tracing.WrapHandlerFunc(corsMiddleware(HospitalPatientsHandler), "/api/v1/hospital/patients"))
	mux.Handle("POST /api/v1/hospital/records/summary", tracing.WrapHandlerFunc(corsMiddleware(HospitalPatientSummaryHandler), "/api/v1/hospital/records/summary"))
	mux.Handle("GET /api/v1/patient/profile", tracing.WrapHandlerFunc(corsMiddleware(PatientProfileHandler), "/api/v1/patient/profile"))
	mux.Handle("GET /api/v1/patient/consents", tracing.WrapHandlerFunc(corsMiddleware(PatientListConsentsHandler), "/api/v1/patient/consents"))
	mux.Handle("POST /api/v1/patient/consents", tracing.WrapHandlerFunc(corsMiddleware(PatientGrantConsentHandler), "/api/v1/patient/consents"))
	mux.Handle("DELETE /api/v1/patient/consents", tracing.WrapHandlerFunc(corsMiddleware(PatientRevokeConsentHandler), "/api/v1/patient/consents"))
	mux.Handle("GET /api/v1/patient/hospital-consents", tracing.WrapHandlerFunc(corsMiddleware(PatientListHospitalConsentsHandler), "/api/v1/patient/hospital-consents"))
	mux.Handle("DELETE /api/v1/patient/hospital-consents/{hospital_id}", tracing.WrapHandlerFunc(corsMiddleware(PatientRevokeHospitalConsentHandler), "/api/v1/patient/hospital-consents/{hospital_id}"))
	mux.Handle("GET /api/v1/patient/consent-requests/{token}", tracing.WrapHandlerFunc(corsMiddleware(PatientLookupConsentRequestHandler), "/api/v1/patient/consent-requests/{token}"))
	mux.Handle("POST /api/v1/patient/consent-requests/{token}/approve", tracing.WrapHandlerFunc(corsMiddleware(PatientApproveConsentRequestHandler), "/api/v1/patient/consent-requests/{token}/approve"))
	mux.Handle("POST /api/v1/patient/records/answer", tracing.WrapHandlerFunc(corsMiddleware(PatientHealthAnswerHandler), "/api/v1/patient/records/answer"))
	mux.Handle("GET /api/v1/patient/calls", tracing.WrapHandlerFunc(corsMiddleware(PatientCallsHandler), "/api/v1/patient/calls"))
	mux.Handle("POST /api/v1/patient/welfare-checks", tracing.WrapHandlerFunc(corsMiddleware(PatientCreateWelfareCheckHandler), "/api/v1/patient/welfare-checks"))
	mux.Handle("GET /api/v1/patient/welfare-checks", tracing.WrapHandlerFunc(corsMiddleware(PatientListWelfareChecksHandler), "/api/v1/patient/welfare-checks"))
	mux.Handle("DELETE /api/v1/patient/welfare-checks/{id}", tracing.WrapHandlerFunc(corsMiddleware(PatientCancelWelfareCheckHandler), "/api/v1/patient/welfare-checks/{id}"))
	mux.Handle("POST /api/v1/patient/calls/bridge-transfer", tracing.WrapHandlerFunc(corsMiddleware(PatientRequestBridgedCallTransferHandler), "/api/v1/patient/calls/bridge-transfer"))
	mux.Handle("GET /api/v1/patient/calls/bridge-session", tracing.WrapHandlerFunc(corsMiddleware(PatientGetBridgedCallSessionHandler), "/api/v1/patient/calls/bridge-session"))
	mux.Handle("PUT /api/v1/patient/calls/bridge-translation", tracing.WrapHandlerFunc(corsMiddleware(PatientUpdateBridgedCallTranslationHandler), "/api/v1/patient/calls/bridge-translation"))
	mux.Handle("POST /api/v1/patient/calls/bridge-end", tracing.WrapHandlerFunc(corsMiddleware(PatientEndBridgedCallHandler), "/api/v1/patient/calls/bridge-end"))
	mux.Handle("GET /api/v1/patient/audit", tracing.WrapHandlerFunc(corsMiddleware(PatientAuditHandler), "/api/v1/patient/audit"))
	mux.Handle("GET /api/v1/hospital/incidents", tracing.WrapHandlerFunc(corsMiddleware(HospitalIncidentsHandler), "/api/v1/hospital/incidents"))
	mux.Handle("GET /api/v1/hospital/patient/audit", tracing.WrapHandlerFunc(corsMiddleware(HospitalPatientAuditHandler), "/api/v1/hospital/patient/audit"))
	mux.Handle("POST /api/v1/hospital/calls/bridge-connect", tracing.WrapHandlerFunc(corsMiddleware(HospitalConnectBridgedCallHandler), "/api/v1/hospital/calls/bridge-connect"))
	mux.Handle("GET /api/v1/hospital/calls/bridge-session", tracing.WrapHandlerFunc(corsMiddleware(HospitalGetBridgedCallSessionHandler), "/api/v1/hospital/calls/bridge-session"))
	mux.Handle("PUT /api/v1/hospital/calls/bridge-translation", tracing.WrapHandlerFunc(corsMiddleware(HospitalUpdateBridgedCallTranslationHandler), "/api/v1/hospital/calls/bridge-translation"))
	mux.Handle("POST /api/v1/hospital/calls/bridge-end", tracing.WrapHandlerFunc(corsMiddleware(HospitalEndBridgedCallHandler), "/api/v1/hospital/calls/bridge-end"))
	mux.Handle("GET /api/v1/hospital/calls/bridge-sessions", tracing.WrapHandlerFunc(corsMiddleware(HospitalListBridgedCallSessionsHandler), "/api/v1/hospital/calls/bridge-sessions"))
	mux.Handle("POST /api/v1/patient/meetings", tracing.WrapHandlerFunc(corsMiddleware(PatientScheduleMeetingHandler), "/api/v1/patient/meetings"))
	mux.Handle("GET /api/v1/patient/meetings", tracing.WrapHandlerFunc(corsMiddleware(PatientListMeetingsHandler), "/api/v1/patient/meetings"))
	mux.Handle("DELETE /api/v1/patient/meetings/{id}", tracing.WrapHandlerFunc(corsMiddleware(PatientCancelMeetingHandler), "/api/v1/patient/meetings/{id}"))
	mux.Handle("GET /api/v1/patient/schedulable-staff", tracing.WrapHandlerFunc(corsMiddleware(PatientListSchedulableStaffHandler), "/api/v1/patient/schedulable-staff"))
	mux.Handle("POST /api/v1/hospital/meetings", tracing.WrapHandlerFunc(corsMiddleware(HospitalScheduleMeetingHandler), "/api/v1/hospital/meetings"))
	mux.Handle("GET /api/v1/hospital/meetings", tracing.WrapHandlerFunc(corsMiddleware(HospitalListMeetingsHandler), "/api/v1/hospital/meetings"))
	mux.Handle("POST /api/v1/hospital/meetings/{id}/accept", tracing.WrapHandlerFunc(corsMiddleware(HospitalAcceptMeetingHandler), "/api/v1/hospital/meetings/{id}/accept"))
	mux.Handle("POST /api/v1/hospital/meetings/{id}/reschedule", tracing.WrapHandlerFunc(corsMiddleware(HospitalRescheduleMeetingHandler), "/api/v1/hospital/meetings/{id}/reschedule"))
	mux.Handle("DELETE /api/v1/hospital/meetings/{id}", tracing.WrapHandlerFunc(corsMiddleware(HospitalCancelMeetingHandler), "/api/v1/hospital/meetings/{id}"))
	mux.Handle("POST /api/v1/hospital/consent-requests", tracing.WrapHandlerFunc(corsMiddleware(HospitalCreateConsentRequestHandler), "/api/v1/hospital/consent-requests"))
	mux.Handle("GET /api/v1/hospital/consent-requests", tracing.WrapHandlerFunc(corsMiddleware(HospitalListConsentRequestsHandler), "/api/v1/hospital/consent-requests"))

	mux.Handle("GET /api/v1/patient/appointment-slots", tracing.WrapHandlerFunc(corsMiddleware(PatientListAppointmentSlotsHandler), "/api/v1/patient/appointment-slots"))
	mux.Handle("GET /api/v1/patient/appointments", tracing.WrapHandlerFunc(corsMiddleware(PatientListAppointmentsHandler), "/api/v1/patient/appointments"))
	mux.Handle("POST /api/v1/patient/appointments", tracing.WrapHandlerFunc(corsMiddleware(PatientBookAppointmentHandler), "/api/v1/patient/appointments"))
	mux.Handle("POST /api/v1/patient/appointments/{id}/reschedule", tracing.WrapHandlerFunc(corsMiddleware(PatientRescheduleAppointmentHandler), "/api/v1/patient/appointments/{id}/reschedule"))
	mux.Handle("DELETE /api/v1/patient/appointments/{id}", tracing.WrapHandlerFunc(corsMiddleware(PatientCancelAppointmentHandler), "/api/v1/patient/appointments/{id}"))
	mux.Handle("GET /api/v1/hospital/appointments", tracing.WrapHandlerFunc(corsMiddleware(HospitalListAppointmentsHandler), "/api/v1/hospital/appointments"))
	mux.Handle("POST /api/v1/hospital/appointments", tracing.WrapHandlerFunc(corsMiddleware(HospitalBookAppointmentHandler), "/api/v1/hospital/appointments"))
	mux.Handle("DELETE /api/v1/hospital/appointments/{id}", tracing.WrapHandlerFunc(corsMiddleware(HospitalCancelAppointmentHandler), "/api/v1/hospital/appointments/{id}"))
	mux.Handle("GET /api/v1/hospital/appointment-slots", tracing.WrapHandlerFunc(corsMiddleware(HospitalListAppointmentSlotsHandler), "/api/v1/hospital/appointment-slots"))
	mux.Handle("GET /api/v1/hospital/availability", tracing.WrapHandlerFunc(corsMiddleware(HospitalGetAvailabilityHandler), "/api/v1/hospital/availability"))
	mux.Handle("PUT /api/v1/hospital/availability", tracing.WrapHandlerFunc(corsMiddleware(HospitalSetAvailabilityHandler), "/api/v1/hospital/availability"))
	mux.Handle("POST /api/v1/hospital/availability/exceptions", tracing.WrapHandlerFunc(corsMiddleware(HospitalAddAvailabilityExceptionHandler), "/api/v1/hospital/availability/exceptions"))
	mux.Handle("DELETE /api/v1/hospital/availability/exceptions/{id}", tracing.WrapHandlerFunc(corsMiddleware(HospitalDeleteAvailabilityExceptionHandler), "/api/v1/hospital/availability/exceptions/{id}"))

	server := &http.Server{
		Addr:    httpAddr,
		Handler: otelhttp.NewHandler(mux, "api-gateway"),
	}

	log.Printf("API Gateway listening on %s", httpAddr)
	if err := server.ListenAndServe(); err != nil {
		log.Printf("Http server error: %v", err)
	}
}
