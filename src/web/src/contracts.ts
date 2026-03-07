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
  HOSPITAL_REGISTER = "/api/v1/auth/hospital/register",
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
