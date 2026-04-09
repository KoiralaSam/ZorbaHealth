package errors

import "errors"

var (
	ErrPendingRegistrationRequestNil    = errors.New("pending registration request is nil")
	ErrPendingRegistrationEmailEmpty    = errors.New("pending registration email is empty")
	ErrVerificationTokenEmpty           = errors.New("verification token is empty")
	ErrPublicWebBaseURLNotConfigured    = errors.New("PUBLIC_WEB_BASE_URL is not configured")
	ErrPhoneNumberEmpty                 = errors.New("phone number is empty")
	ErrOTPEmpty                         = errors.New("otp is empty")

	ErrPendingVerificationEventMissingRegisterRequest = errors.New("pending verification event missing register_request")

	ErrSendGridSendFailed = errors.New("sendgrid: status=%d body=%s")

	ErrVoipmsDIDNotSet             = errors.New("voipms: VOIPMS_DID is not set")
	ErrVoipmsToPhoneNumberEmpty  = errors.New("voipms: to phone number is empty")
	ErrVoipmsMessageEmpty         = errors.New("voipms: message is empty")
	ErrVoipmsAPIUsernameRequired = errors.New("voipms: api_username and api_password are required")

	ErrVoipmsNewRequest = errors.New("voipms: new request: %w")
	ErrVoipmsDoRequest  = errors.New("voipms: do request: %w")

	ErrVoipmsStatusBody = errors.New("voipms: status=%d body=%s")
	ErrVoipmsUnexpectedResponse = errors.New("voipms: unexpected response: %s")
	ErrVoipmsAPIStatusMsg       = errors.New("voipms: api status=%s msg=%s")
)

