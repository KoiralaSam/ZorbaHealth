package http

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

// parseInboundSMS reads VoIP.ms SMS URL callback data.
// VoIP.ms normally issues GET with query parameters (id, date, from/contact, did, message).
// Some setups use POST with application/x-www-form-urlencoded or JSON.
func parseInboundSMS(r *http.Request) (SMSRequest, error) {
	if r.Method == http.MethodGet {
		return smsFromValues(r.URL.Query()), nil
	}

	ct := r.Header.Get("Content-Type")
	if ct == "application/x-www-form-urlencoded" || ct == "multipart/form-data" {
		if err := r.ParseForm(); err != nil {
			return SMSRequest{}, err
		}
		return smsFromValues(r.PostForm), nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return SMSRequest{}, err
	}
	if len(body) == 0 {
		return SMSRequest{}, nil
	}
	var req SMSRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return SMSRequest{}, err
	}
	return req, nil
}

func smsFromValues(v url.Values) SMSRequest {
	get := func(keys ...string) string {
		for _, k := range keys {
			if s := v.Get(k); s != "" {
				return s
			}
		}
		return ""
	}
	return SMSRequest{
		ID:          get("id"),
		Date:        get("date"),
		From:        get("from", "contact", "sender"),
		To:          get("to", "did"),
		PhoneNumber: get("phone_number", "phone"),
		Message:     get("message", "msg", "text"),
		Files:       get("files"),
	}
}
