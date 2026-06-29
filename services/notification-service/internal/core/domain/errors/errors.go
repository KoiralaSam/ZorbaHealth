package errors

import "errors"

var (
	ErrPendingRegistrationRequestNil = errors.New("pending registration request is nil")
	ErrPendingRegistrationEmailEmpty = errors.New("pending registration email is empty")
	ErrVerificationTokenEmpty        = errors.New("verification token is empty")
	ErrPublicWebBaseURLNotConfigured = errors.New("PUBLIC_WEB_BASE_URL is not configured")
	ErrPhoneNumberEmpty              = errors.New("phone number is empty")
	ErrOTPEmpty                      = errors.New("otp is empty")

	ErrPendingVerificationEventMissingRegisterRequest = errors.New("pending verification event missing register_request")

	ErrMailtrapSendFailed = errors.New("mailtrap send failed")

	ErrVoipmsDIDNotSet           = errors.New("voipms: VOIPMS_DID is not set")
	ErrVoipmsToPhoneNumberEmpty  = errors.New("voipms: to phone number is empty")
	ErrVoipmsMessageEmpty        = errors.New("voipms: message is empty")
	ErrVoipmsAPIUsernameRequired = errors.New("voipms: api_username and api_password are required")
	ErrUnsupportedEmailProvider  = errors.New("unsupported email provider")
	ErrUnsupportedSMSProvider    = errors.New("unsupported sms provider")
)
