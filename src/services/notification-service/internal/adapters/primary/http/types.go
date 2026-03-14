package http

// SMSRequest is the JSON body for the VoIP.ms incoming SMS webhook.
type SMSRequest struct {
	PhoneNumber string `json:"phone_number"`
	Message     string `json:"message"`
}
