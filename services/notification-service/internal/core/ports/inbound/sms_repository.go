package inbound

import "context"

type SMSReceiver interface {
	ReceiveSMS(ctx context.Context, phoneNumber, message string) error
}
