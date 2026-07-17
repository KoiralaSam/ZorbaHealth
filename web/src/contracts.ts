import { APIResponse } from "./types";

// API Endpoints that are currently implemented in the API Gateway
export enum APIEndpoints {
  // Health & Info
  HEALTH = "/health",
  ROOT = "/",

  // Authentication
  PATIENT_LOGIN = "/api/v1/auth/patient/login",
  HOSPITAL_LOGIN = "/api/v1/auth/hospital/login",

  // Patient Registration
  PATIENT_REGISTER = "/api/v1/auth/patient/register",
  PATIENT_REGISTER_VERIFY = "/api/v1/auth/patient/register/verify",
  PATIENT_REGISTER_VERIFY_OTP = "/api/v1/auth/patient/register/verify-otp",
  PATIENT_PROFILE = "/api/v1/patient/profile",
  PATIENT_CONSENTS = "/api/v1/patient/consents",
  PATIENT_RECORDS_ANSWER = "/api/v1/patient/records/answer",
  PATIENT_CALLS = "/api/v1/patient/calls",
  PATIENT_WELFARE_CHECKS = "/api/v1/patient/welfare-checks",
  PATIENT_AUDIT = "/api/v1/patient/audit",
  HOSPITAL_REGISTER = "/api/v1/auth/hospital/register",
  HOSPITAL_PATIENT_SUMMARY = "/api/v1/hospital/records/summary",
  HOSPITAL_INCIDENTS = "/api/v1/hospital/incidents",
  HOSPITAL_PATIENT_AUDIT = "/api/v1/hospital/patient/audit",
}

// HTTP Methods
export enum HTTPMethod {
  GET = "GET",
  POST = "POST",
  PUT = "PUT",
  DELETE = "DELETE",
  PATCH = "PATCH",
}

// API Error Codes (matching backend error codes)
export enum ErrorCode {
  METHOD_NOT_ALLOWED = "METHOD_NOT_ALLOWED",
  INVALID_REQUEST_BODY = "INVALID_REQUEST_BODY",
  UNAUTHORIZED = "UNAUTHORIZED",
  FORBIDDEN = "FORBIDDEN",
  NOT_FOUND = "NOT_FOUND",
  INTERNAL_SERVER_ERROR = "INTERNAL_SERVER_ERROR",
}

// =============================================================================
// HTTP Request/Response Payloads
// =============================================================================

// Health Check - GET /health
export interface HTTPHealthCheckResponse {
  status: string;
  service: string;
}

// Root - GET /
export interface HTTPRootResponse {
  message: string;
  version: string;
}

// Patient Login - POST /api/v1/auth/patient/login
export interface HTTPPatientLoginRequest {
  phone_number: string;
  password: string;
  email?: string;
  full_name?: string;
  date_of_birth?: string;
}

export interface PatientLoginResponseData {
  message?: string;
  access_token?: string;
  patient_id?: string;
}

export type HTTPPatientLoginResponse = APIResponse<PatientLoginResponseData>;

export interface PatientProfileResponseData {
  patient_id?: string;
  full_name?: string;
  email?: string;
  phone_number?: string;
  date_of_birth?: string;
  medical_notes?: string;
  voice_phone?: string;
  voice_enabled?: boolean;
  support_window?: string;
}

export type HTTPPatientProfileResponse = APIResponse<PatientProfileResponseData>;

export interface ConsentRecord {
  consent_id?: string;
  consent_type?: string;
  granted_by?: string;
  granted_at?: string;
  revoked_at?: string;
  scope?: string;
  expiration_time?: string;
  source?: string;
  status?: string;
  metadata?: Record<string, unknown>;
}

export interface PatientConsentListResponseData {
  patient_id?: string;
  consents?: ConsentRecord[];
}

export type HTTPPatientConsentListResponse =
  APIResponse<PatientConsentListResponseData>;

export interface HTTPPatientConsentMutationRequest {
  consent_type: string;
  scope?: string;
  source?: string;
  metadata?: Record<string, unknown>;
}

export interface PatientConsentMutationResponseData {
  message?: string;
  consent?: ConsentRecord;
}

export type HTTPPatientConsentMutationResponse =
  APIResponse<PatientConsentMutationResponseData>;

export interface HTTPPatientHealthAnswerRequest {
  question: string;
  top_k?: number;
}

export interface PatientHealthCitation {
  text?: string;
  source_file?: string;
  score?: number;
}

export interface PatientHealthAnswerResponseData {
  answer?: string;
  citations?: PatientHealthCitation[];
}

export type HTTPPatientHealthAnswerResponse =
  APIResponse<PatientHealthAnswerResponseData>;

export interface PatientCallSummary {
  id: number;
  status?: string;
  started_at?: string;
  ended_at?: string;
  summary?: string;
  recording_url?: string;
  livekit_room_id?: string;
}

export interface PatientCallListResponseData {
  patient_id?: string;
  calls?: PatientCallSummary[];
}

export type HTTPPatientCallListResponse =
  APIResponse<PatientCallListResponseData>;

export type WelfareCheckReason =
  | "medication_reminder"
  | "mental_wellbeing"
  | "daily_checkup"
  | "symptom_follow_up"
  | "care_plan_reminder"
  | "other";

export interface HTTPPatientWelfareCheckCreateRequest {
  scheduled_at: string;
  timezone: string;
  reason_code: WelfareCheckReason;
  reason_detail?: string;
}

export interface PatientWelfareCheck {
  id?: string;
  patient_id?: string;
  scheduled_at?: string;
  timezone?: string;
  reason_code?: WelfareCheckReason | string;
  reason_detail?: string;
  status?: string;
  attempt_count?: number;
  latest_run_id?: string;
  latest_run_status?: string;
  latest_run_attempts?: number;
  latest_run_failure_reason?: string;
  livekit_room_id?: string;
  sip_participant_id?: string;
  created_at?: string;
  updated_at?: string;
  cancelled_at?: string;
}

export interface PatientWelfareCheckResponseData {
  welfare_check?: PatientWelfareCheck;
}

export interface PatientWelfareCheckListResponseData {
  welfare_checks?: PatientWelfareCheck[];
}

export type HTTPPatientWelfareCheckResponse =
  APIResponse<PatientWelfareCheckResponseData>;

export type HTTPPatientWelfareCheckListResponse =
  APIResponse<PatientWelfareCheckListResponseData>;

export interface AuditEventRecord {
  event_id?: string;
  event_type?: string;
  actor_type?: string;
  actor_id?: string;
  patient_id?: string;
  service_name?: string;
  resource_type?: string;
  resource_id?: string;
  timestamp?: string;
  correlation_id?: string;
  tool_name?: string;
  success_status?: boolean;
  failure_reason?: string;
  metadata?: Record<string, unknown>;
}

export interface PatientAuditResponseData {
  patient_id?: string;
  events?: AuditEventRecord[];
}

export type HTTPPatientAuditResponse = APIResponse<PatientAuditResponseData>;

// Patient Register - POST /api/v1/auth/patient/register
export interface HTTPPatientRegisterRequest {
  phone_number: string;
  password: string;
  email?: string;
  full_name?: string;
  date_of_birth?: string; // Date string in YYYY-MM-DD format
}

export interface PatientRegisterResponseData {
  message?: string;
  patient_id?: string;
}

export type HTTPPatientRegisterResponse =
  APIResponse<PatientRegisterResponseData>;

// Patient Verify OTP - POST /api/v1/auth/patient/register/verify-otp
export interface HTTPPatientVerifyOTPRequest {
  phone_number: string;
  otp: string;
}

export interface PatientVerifyOTPResponseData {
  message?: string;
}

export type HTTPPatientVerifyOTPResponse =
  APIResponse<PatientVerifyOTPResponseData>;

// Hospital Login - POST /api/v1/auth/hospital/login
export interface HTTPHospitalLoginRequest {
  email: string;
  password: string;
}

export interface HospitalLoginResponseData {
  message?: string;
  access_token?: string;
  hospital_id?: string;
  role?: string;
}

export type HTTPHospitalLoginResponse = APIResponse<HospitalLoginResponseData>;

export interface HTTPHospitalPatientSummaryRequest {
  patient_id: string;
  focus?: string;
}

export interface HospitalPatientSummaryResponseData {
  summary?: string;
}

export type HTTPHospitalPatientSummaryResponse =
  APIResponse<HospitalPatientSummaryResponseData>;

export interface HospitalIncidentRecord {
  event_id?: string;
  patient_id?: string;
  timestamp?: string;
  severity?: string;
  session_id?: string;
  service_name?: string;
  failure_reason?: string;
  metadata?: Record<string, unknown>;
}

export interface HospitalIncidentListResponseData {
  incidents?: HospitalIncidentRecord[];
}

export type HTTPHospitalIncidentListResponse =
  APIResponse<HospitalIncidentListResponseData>;

export interface HospitalPatientAuditResponseData {
  patient_id?: string;
  events?: AuditEventRecord[];
}

export type HTTPHospitalPatientAuditResponse =
  APIResponse<HospitalPatientAuditResponseData>;

// =============================================================================
// Type Guards & Validators
// =============================================================================

/**
 * Validates if the given string is a valid API endpoint
 */
export function isValidEndpoint(endpoint: string): endpoint is APIEndpoints {
  return Object.values(APIEndpoints).includes(endpoint as APIEndpoints);
}

/**
 * Validates if the given string is a valid HTTP method
 */
export function isValidHTTPMethod(method: string): method is HTTPMethod {
  return Object.values(HTTPMethod).includes(method as HTTPMethod);
}

/**
 * Validates if the given string is a valid error code
 */
export function isValidErrorCode(code: string): code is ErrorCode {
  return Object.values(ErrorCode).includes(code as ErrorCode);
}
