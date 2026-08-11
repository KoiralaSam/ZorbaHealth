import { APIResponse } from "./types";

// API Endpoints that are currently implemented in the API Gateway
export enum APIEndpoints {
  // Health & Info
  HEALTH = "/health",
  ROOT = "/",

  // Authentication
  PATIENT_LOGIN = "/api/v1/auth/patient/login",
  PATIENT_REFRESH = "/api/v1/auth/patient/refresh",
  PATIENT_LOGOUT = "/api/v1/auth/patient/logout",
  HOSPITAL_LOGIN = "/api/v1/auth/hospital/login",
  HOSPITAL_REFRESH = "/api/v1/auth/hospital/refresh",
  HOSPITAL_LOGOUT = "/api/v1/auth/hospital/logout",
  HOSPITAL_STAFF_REGISTER = "/api/v1/auth/hospital/staff/register",
  HOSPITAL_PATIENTS = "/api/v1/hospital/patients",

  // Patient Registration
  PATIENT_REGISTER = "/api/v1/auth/patient/register",
  PATIENT_REGISTER_VERIFY = "/api/v1/auth/patient/register/verify",
  PATIENT_REGISTER_VERIFY_OTP = "/api/v1/auth/patient/register/verify-otp",
  PATIENT_PROFILE = "/api/v1/patient/profile",
  PATIENT_CONSENTS = "/api/v1/patient/consents",
  PATIENT_HOSPITAL_CONSENTS = "/api/v1/patient/hospital-consents",
  PATIENT_CONSENT_REQUESTS = "/api/v1/patient/consent-requests",
  PATIENT_RECORDS_ANSWER = "/api/v1/patient/records/answer",
  PATIENT_CALLS = "/api/v1/patient/calls",
  PATIENT_WELFARE_CHECKS = "/api/v1/patient/welfare-checks",
  PATIENT_BRIDGED_CALL_TRANSFER = "/api/v1/patient/calls/bridge-transfer",
  PATIENT_BRIDGED_CALL_SESSION = "/api/v1/patient/calls/bridge-session",
  PATIENT_BRIDGED_CALL_TRANSLATION = "/api/v1/patient/calls/bridge-translation",
  PATIENT_BRIDGED_CALL_END = "/api/v1/patient/calls/bridge-end",
  PATIENT_AUDIT = "/api/v1/patient/audit",
  PATIENT_MEETINGS = "/api/v1/patient/meetings",
  PATIENT_SCHEDULABLE_STAFF = "/api/v1/patient/schedulable-staff",
  PATIENT_APPOINTMENTS = "/api/v1/patient/appointments",
  PATIENT_APPOINTMENT_SLOTS = "/api/v1/patient/appointment-slots",
  HOSPITAL_REGISTER = "/api/v1/auth/hospital/register",
  HOSPITAL_PATIENT_SUMMARY = "/api/v1/hospital/records/summary",
  HOSPITAL_INCIDENTS = "/api/v1/hospital/incidents",
  HOSPITAL_PATIENT_AUDIT = "/api/v1/hospital/patient/audit",
  HOSPITAL_BRIDGED_CALL_CONNECT = "/api/v1/hospital/calls/bridge-connect",
  HOSPITAL_BRIDGED_CALL_SESSION = "/api/v1/hospital/calls/bridge-session",
  HOSPITAL_BRIDGED_CALL_TRANSLATION = "/api/v1/hospital/calls/bridge-translation",
  HOSPITAL_BRIDGED_CALL_END = "/api/v1/hospital/calls/bridge-end",
  HOSPITAL_BRIDGED_CALL_SESSIONS = "/api/v1/hospital/calls/bridge-sessions",
  HOSPITAL_MEETINGS = "/api/v1/hospital/meetings",
  HOSPITAL_APPOINTMENTS = "/api/v1/hospital/appointments",
  HOSPITAL_APPOINTMENT_SLOTS = "/api/v1/hospital/appointment-slots",
  HOSPITAL_AVAILABILITY = "/api/v1/hospital/availability",
  HOSPITAL_AVAILABILITY_EXCEPTIONS = "/api/v1/hospital/availability/exceptions",
  HOSPITAL_CONSENT_REQUESTS = "/api/v1/hospital/consent-requests",
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
  identifier: string;
  password: string;
  phone_number?: string;
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

export interface HospitalConsentRequestRecord {
  id?: string;
  token?: string;
  hospital_id?: string;
  hospital_name?: string;
  staff_id?: string;
  staff_name?: string;
  staff_role?: string;
  patient_id?: string;
  requested_permissions?: string[];
  note?: string;
  expires_at?: string;
  approved_at?: string;
  created_at?: string;
  status?: string;
  qr_payload?: string;
}

export interface HTTPHospitalConsentRequestCreateRequest {
  requested_permissions?: string[];
  note?: string;
  expires_in_minutes?: number;
}

export interface HospitalConsentRequestResponseData {
  request?: HospitalConsentRequestRecord;
}

export type HTTPHospitalConsentRequestCreateResponse =
  APIResponse<HospitalConsentRequestResponseData>;

export interface HospitalConsentRequestListResponseData {
  requests?: HospitalConsentRequestRecord[];
}

export type HTTPHospitalConsentRequestListResponse =
  APIResponse<HospitalConsentRequestListResponseData>;

export interface PatientHospitalConsentRecord {
  hospital_id?: string;
  hospital_name?: string;
  granted_at?: string;
  revoked_at?: string;
  status?: string;
}

export interface PatientHospitalConsentListResponseData {
  consents?: PatientHospitalConsentRecord[];
}

export type HTTPPatientHospitalConsentListResponse =
  APIResponse<PatientHospitalConsentListResponseData>;

export type HTTPPatientConsentRequestLookupResponse =
  APIResponse<HospitalConsentRequestResponseData>;

export interface PatientConsentRequestApproveResponseData {
  message?: string;
  consent?: PatientHospitalConsentRecord;
}

export type HTTPPatientConsentRequestApproveResponse =
  APIResponse<PatientConsentRequestApproveResponseData>;

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

export interface BridgedCallTranslationPreferencesRecord {
  enabled?: boolean;
  language_mode?: string;
  language_code?: string;
  participant_identity?: string;
  updated_at?: string;
}

export interface BridgedCallSessionRecord {
  session_id?: string;
  room_sid?: string;
  patient_id?: string;
  hospital_id?: string;
  staff_id?: string;
  status?: string;
  requested_by_actor_type?: string;
  requested_by_actor_id?: string;
  transfer_reason?: string;
  requested_at?: string;
  connected_at?: string;
  ended_at?: string;
  patient_translation?: BridgedCallTranslationPreferencesRecord;
  staff_translation?: BridgedCallTranslationPreferencesRecord;
}

export interface RequestBridgedCallTransferRequest {
  session_id: string;
  room_sid?: string;
  hospital_id: string;
  staff_id?: string;
  transfer_reason?: string;
}

export interface ConnectBridgedCallRequest {
  session_id: string;
  staff_participant_identity?: string;
  /** web (default) joins LiveKit in-browser; phone dials the staff PSTN number via LiveKit SIP */
  join_mode?: "web" | "phone";
}

export interface UpdateBridgedCallTranslationRequest {
  session_id: string;
  participant: "patient" | "staff";
  translation: BridgedCallTranslationPreferencesRecord;
}

export interface EndBridgedCallRequest {
  session_id: string;
  reason?: string;
}

export interface BridgedCallSessionResponseData {
  session?: BridgedCallSessionRecord;
  patient_room_token?: string;
  livekit_ws_url?: string;
}

export type HTTPBridgedCallSessionResponse =
  APIResponse<BridgedCallSessionResponseData>;

export interface BridgedCallConnectResponseData {
  session?: BridgedCallSessionRecord;
  staff_room_token?: string;
  livekit_ws_url?: string;
  patient_room_token?: string;
}

export type HTTPBridgedCallConnectResponse =
  APIResponse<BridgedCallConnectResponseData>;

export interface BridgedCallSessionListResponseData {
  sessions?: BridgedCallSessionRecord[];
}

export type HTTPBridgedCallSessionListResponse =
  APIResponse<BridgedCallSessionListResponseData>;

// Payload published by the voice agent on LiveKit data topic "zorba.interpretation".
export interface InterpretationSegmentMessage {
  type?: string;
  session_id?: string;
  participant?: string;
  original_text?: string;
  translated_text?: string;
  target_language?: string;
  passthrough?: boolean;
}

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

export const AUDIT_EVENT_TYPES = [
  "PATIENT_CREATED",
  "PATIENT_VERIFIED",
  "PATIENT_LOGIN",
  "PATIENT_LOGOUT",
  "HEALTH_RECORD_CREATED",
  "HEALTH_RECORD_VIEWED",
  "HEALTH_RECORD_SEARCHED",
  "HEALTH_RECORD_SUMMARIZED",
  "AI_TOOL_CALLED",
  "AI_RESPONSE_GENERATED",
  "LOCATION_REQUESTED",
  "EMERGENCY_ESCALATION_TRIGGERED",
  "NOTIFICATION_SENT",
  "CONSENT_GRANTED",
  "CONSENT_REVOKED",
  "TRANSLATION_REQUESTED",
  "CALL_TRANSFER_REQUESTED",
  "CALL_TRANSFER_CONNECTED",
  "CALL_BRIDGED_ENDED",
  "INTERPRETATION_SESSION_STARTED",
  "INTERPRETATION_SESSION_ENDED",
  "INTERPRETATION_PREFERENCES_UPDATED",
  "INTERPRETATION_SEGMENT_PROCESSED",
] as const;

export function getAuditEventTypeOptions(events: AuditEventRecord[]) {
  return Array.from(
    new Set([
      ...AUDIT_EVENT_TYPES,
      ...events.map((event) => event.event_type).filter(Boolean),
    ]),
  ) as string[];
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
  staff_id?: string;
  role?: string;
}

export type HTTPHospitalLoginResponse = APIResponse<HospitalLoginResponseData>;

export interface HTTPHospitalRegisterRequest {
  hospital_name: string;
  license_no: string;
  email: string;
  phone_number?: string;
  password: string;
  staff_name: string;
  staff_role?: string;
  address?: string;
}

export interface HospitalRegisterResponseData {
  message?: string;
  user_id?: string;
  hospital_id?: string;
  staff_id?: string;
  staff_role?: string;
}

export type HTTPHospitalRegisterResponse =
  APIResponse<HospitalRegisterResponseData>;

export interface HTTPHospitalStaffRegisterRequest {
  email: string;
  phone_number?: string;
  password: string;
  staff_name: string;
  staff_role: string;
}

export type HTTPHospitalStaffRegisterResponse =
  APIResponse<HospitalRegisterResponseData>;

export interface HTTPHospitalPatientSummaryRequest {
  patient_id: string;
  focus?: string;
}

export interface HospitalPatientSummaryResponseData {
  patient_id?: string;
  summary?: string;
}

export type HTTPHospitalPatientSummaryResponse =
  APIResponse<HospitalPatientSummaryResponseData>;

export interface HospitalPatientRecord {
  patient_id?: string;
  full_name?: string;
  email?: string;
  phone_number?: string;
  date_of_birth?: string;
  consent_granted_at?: string;
  last_call_at?: string;
}

export interface HospitalPatientListResponseData {
  patients?: HospitalPatientRecord[];
}

export type HTTPHospitalPatientListResponse =
  APIResponse<HospitalPatientListResponseData>;

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

export interface HospitalMeetingRecord {
  id: string;
  patient_id?: string;
  staff_id?: string;
  hospital_id?: string;
  starts_at?: string;
  duration_minutes?: number;
  timezone?: string;
  title?: string;
  join_url?: string;
  livekit_room_name?: string;
  livekit_server_url?: string;
  participant_token?: string;
  status?: string;
  correlation_id?: string;
}

export interface HospitalMeetingListResponseData {
  meetings?: HospitalMeetingRecord[];
}

export type HTTPHospitalMeetingListResponse =
  APIResponse<HospitalMeetingListResponseData>;

export interface HTTPHospitalMeetingScheduleRequest {
  patient_id: string;
  staff_id?: string;
  hospital_id?: string;
  starts_at: string;
  duration_minutes: number;
  timezone: string;
  title?: string;
  notes?: string;
  send_sms?: boolean;
}

export interface HTTPHospitalMeetingRescheduleRequest {
  starts_at: string;
  duration_minutes: number;
  timezone: string;
  title?: string;
}

export interface HTTPHospitalMeetingCancelRequest {
  reason?: string;
}

export interface HospitalMeetingMutationResponseData {
  meeting?: HospitalMeetingRecord;
}

export type HTTPHospitalMeetingMutationResponse =
  APIResponse<HospitalMeetingMutationResponseData>;

// =============================================================================
// Patient Meeting Types
// =============================================================================

export interface PatientMeetingRecord {
  id: string;
  patient_id?: string;
  staff_id?: string;
  hospital_id?: string;
  starts_at?: string;
  duration_minutes?: number;
  timezone?: string;
  title?: string;
  join_url?: string;
  livekit_room_name?: string;
  livekit_server_url?: string;
  participant_token?: string;
  status?: string;
  correlation_id?: string;
}

export interface HTTPPatientMeetingScheduleRequest {
  staff_id: string;
  hospital_id: string;
  starts_at: string;
  duration_minutes: number;
  timezone: string;
  title?: string;
  notes?: string;
  send_sms?: boolean;
}

export type HTTPPatientMeetingListResponse = APIResponse<HospitalMeetingListResponseData>;

export type HTTPPatientMeetingMutationResponse = APIResponse<HospitalMeetingMutationResponseData>;

export interface PatientSchedulableStaffRecord {
  staff_id: string;
  hospital_id: string;
  name: string;
  role: string;
  email: string;
}

export interface PatientSchedulableStaffListResponseData {
  staff?: PatientSchedulableStaffRecord[];
}

export type HTTPPatientSchedulableStaffResponse =
  APIResponse<PatientSchedulableStaffListResponseData>;

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
