package main

import "time"

type PatientLoginRequest struct {
	PhoneNumber string `json:"phone_number"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type PatientLoginResponse struct {
	Message     string `json:"message,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	PatientID   string `json:"patient_id,omitempty"`
}

type PatientRegisterRequest struct {
	PhoneNumber string    `json:"phone_number"`
	Email       string    `json:"email"`
	Password    string    `json:"password"`
	FullName    string    `json:"full_name"`
	DateOfBirth time.Time `json:"date_of_birth"`
}

type PatientRegisterResponse struct {
	Message   string `json:"message,omitempty"`
	PatientID string `json:"patient_id,omitempty"`
}

type PatientRegisterVerifyOTPRequest struct {
	PhoneNumber string `json:"phone_number"`
	OTP         string `json:"otp"`
}

type HospitalLoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type HospitalLoginResponse struct {
	Message     string `json:"message,omitempty"`
	AccessToken string `json:"access_token,omitempty"`
	HospitalID  string `json:"hospital_id,omitempty"`
	Role        string `json:"role,omitempty"`
}
