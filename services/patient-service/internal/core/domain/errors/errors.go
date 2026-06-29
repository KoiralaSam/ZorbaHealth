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

	ErrInvalidPhoneNumberNoDigits = errors.New("invalid phone number: no digits")
	ErrEmailRequired              = errors.New("email is required")

	ErrMeetingStaffNotFound       = errors.New("staff member not found")
	ErrMeetingStaffInactive       = errors.New("staff member is not active")
	ErrMeetingInvalidRole         = errors.New("staff role cannot be scheduled for visits")
	ErrMeetingHospitalMismatch    = errors.New("staff does not belong to hospital")
	ErrMeetingConsentRequired     = errors.New("patient has not consented to share data with this hospital")
	ErrMeetingStartsAtInvalid     = errors.New("meeting must be scheduled in the future")
	ErrMeetingDurationInvalid     = errors.New("meeting duration is invalid")
	ErrMeetingNotFound            = errors.New("meeting not found")
	ErrMeetingAlreadyCancelled    = errors.New("meeting is already cancelled")
	ErrMeetingNotPending          = errors.New("meeting is not pending staff approval")
	ErrMeetingNotificationConsent = errors.New("email notification consent is required")
	ErrMeetingPatientEmailMissing = errors.New("a verified email address is required on your patient profile to schedule visits")
	ErrMeetingLiveKitUnavailable  = errors.New("livekit meeting provider is unavailable")

	ErrBridgedCallSessionRequired    = errors.New("bridged call session_id is required")
	ErrBridgedCallHospitalRequired   = errors.New("hospital_id is required to transfer a call")
	ErrBridgedCallNotFound           = errors.New("bridged call session not found")
	ErrBridgedCallForbidden          = errors.New("bridged call access is forbidden")
	ErrBridgedCallAlreadyEnded       = errors.New("bridged call session already ended")
	ErrBridgedCallInvalidParticipant = errors.New("bridged call participant must be patient or staff")
	ErrBridgedCallInvalidMode        = errors.New("translation mode must be auto or manual")
	ErrBridgedCallStoreUnavailable   = errors.New("bridged call session store is unavailable")
)
