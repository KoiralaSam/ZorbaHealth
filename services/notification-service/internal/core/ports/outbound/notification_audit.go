package outbound

import "context"

type NotificationAudit interface {
	RecordNotificationSent(ctx context.Context, patientID, correlationID, meetingID, channel, recipientRole, template string, success bool, failureReason string) error
}
