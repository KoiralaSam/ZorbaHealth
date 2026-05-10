package outbound

import "context"

type SMSSender interface {
	SendSMS(ctx context.Context, toPhoneNumber, message string) error
}
