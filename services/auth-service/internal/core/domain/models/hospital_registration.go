package models

type HospitalRegistration struct {
	HospitalName string
	LicenseNo    string
	StaffName    string
	StaffEmail   string
	StaffPhone   string
	StaffRole    string
	PasswordHash string
}

type HospitalStaffRegistration struct {
	HospitalID   string
	StaffName    string
	StaffEmail   string
	StaffPhone   string
	StaffRole    string
	PasswordHash string
}

type HospitalStaffAccount struct {
	UserID     string
	HospitalID string
	StaffID    string
	StaffRole  string
}
