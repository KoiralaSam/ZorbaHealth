package rag

import (
	"context"
	"fmt"

	sharedaudit "github.com/KoiralaSam/ZorbaHealth/shared/audit"
	auditpb "github.com/KoiralaSam/ZorbaHealth/shared/proto/audit"
)

type GRPCConsentChecker struct {
	client auditpb.AuditServiceClient
}

func NewGRPCConsentChecker(client auditpb.AuditServiceClient) *GRPCConsentChecker {
	return &GRPCConsentChecker{client: client}
}

func (c *GRPCConsentChecker) CheckConsent(ctx context.Context, patientID, consentType string) (bool, string, error) {
	if c.client == nil {
		return true, "", nil
	}
	resp, err := c.client.CheckConsent(ctx, &auditpb.CheckConsentRequest{
		PatientId:   patientID,
		ConsentType: consentType,
	})
	if err != nil {
		return false, "", err
	}
	if resp.GetAllowed() {
		return true, "", nil
	}
	reason := resp.GetDenialReason()
	if reason == "" {
		reason = "consent not granted"
	}
	if consentType == sharedaudit.ConsentHealthRecordAccess {
		return false, reason, nil
	}
	return false, reason, nil
}

func (c *GRPCConsentChecker) String() string {
	return fmt.Sprintf("audit-consent-checker")
}
