// Generic API Response wrapper matching backend types.APIResponse
export interface APIResponse<T> {
  data?: T;
  error?: APIError;
}

// Error structure matching backend types.Error
export interface APIError {
  code: string;
  message: string;
}
