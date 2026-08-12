package models

type HospitalRegistration struct {
	HospitalName string
	LicenseNo    string
	StaffName    string
	StaffEmail   string
	StaffPhone   string
	StaffRole    string
	Password     string
}

type HospitalStaffRegistration struct {
	HospitalID string
	StaffName  string
	StaffEmail string
	StaffPhone string
	StaffRole  string
	Password   string
}

type HospitalStaffAccount struct {
	UserID     string
	HospitalID string
	StaffID    string
	StaffRole  string
}

type CreateHospitalStaffInput struct {
	HospitalName string // empty when attaching to existing hospital
	LicenseNo    string // required with HospitalName
	HospitalID   string // set when attaching to existing hospital
	UserID       string
	StaffName    string
	StaffEmail   string
	StaffPhone   string
	StaffRole    string
}
