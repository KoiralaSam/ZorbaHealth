package errors

import "errors"

var (
	ErrRegistrationRequestRequired = errors.New("registration request is required")
	ErrInvalidPhoneNumber          = errors.New("invalid phone number: must be 10–15 digits, optional leading +")
	ErrDateOfBirthRequired         = errors.New("date of birth is required")
	ErrDateOfBirthInFuture         = errors.New("date of birth cannot be in the future")

	ErrGenerateOTPFailed               = errors.New("failed to generate OTP: ")
	ErrPendingRegistrationSetFailed    = errors.New("failed to set pending registration: ")
	ErrOTPSetFailed                    = errors.New("failed to set OTP: ")
	ErrPublishPatientCachedEventFailed = errors.New("failed to publish patient cached event: ")

	ErrInvalidOrExpiredVerificationLink    = errors.New("invalid or expired verification link: ")
	ErrPendingRegistrationUpdateFailed     = errors.New("failed to update pending registration: ")
	ErrPhoneVerificationRequired           = errors.New("verify your phone to complete registration")
	ErrAuthServiceRegisterPatientFailed    = errors.New("failed to create user in auth service: ")
	ErrAuthServiceInvalidUserID            = errors.New("invalid user_id from auth service: ")
	ErrPatientCreationFailed               = errors.New("failed to create patient: ")
	ErrPublishPatientRegisteredEventFailed = errors.New("failed to publish patient registered event: ")

	ErrInvalidOrExpiredOTP                  = errors.New("invalid or expired OTP")
	ErrInvalidOTPCode                       = errors.New("invalid OTP code")
	ErrPendingRegistrationNotFoundOrExpired = errors.New("pending registration not found or expired")
	ErrExistingPatientNotFound              = errors.New("patient not found for phone number")
	ErrExistingPatientVerificationState     = errors.New("existing patient verification state not found or expired")
	ErrAmbiguousPhoneNumber                 = errors.New("multiple patients found for phone number")
	ErrPhoneNumberAlreadyRegistered         = errors.New("phone number is already registered")

	ErrInvalidPhoneNumberNoDigits = errors.New("invalid phone number: no digits")
	ErrEmailRequired              = errors.New("email is required")
)
