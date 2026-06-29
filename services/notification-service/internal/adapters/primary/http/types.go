package http

// SMSRequest is the VoIP.ms SMS/MMS URL callback payload (GET query, form POST, or JSON POST).
// See shared/types.SMSRequest for the canonical field list used across the repo.
type SMSRequest struct {
	ID          string `json:"id"`
	Date        string `json:"date"`
	From        string `json:"from"`
	To          string `json:"to"`
	PhoneNumber string `json:"phone_number"`
	Message     string `json:"message"`
	Files       string `json:"files"`
}

func (r SMSRequest) SenderPhone() string {
	if r.From != "" {
		return r.From
	}
	return r.PhoneNumber
}
