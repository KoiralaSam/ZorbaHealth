INSERT INTO audit.audit_event_types (event_type, retention_class, description) VALUES
    ('VOICE_OTP_WAIT_REGISTERED', 'compliance', 'Voice session registered an OTP wait for inbound SMS verification.'),
    ('VOICE_INBOUND_SMS_PROCESSED', 'compliance', 'Inbound SMS during an active voice OTP wait was processed.'),
    ('VOICE_OTP_VERIFY_FAILED', 'security', 'Voice OTP verification failed.'),
    ('CALL_TRANSFER_REQUESTED', 'clinical_ops', 'Bridged call transfer to staff was requested.'),
    ('CALL_TRANSFER_CONNECTED', 'clinical_ops', 'Bridged call transfer to staff connected.'),
    ('CALL_BRIDGED_ENDED', 'clinical_ops', 'Bridged call session ended.'),
    ('INTERPRETATION_SESSION_STARTED', 'clinical_ops', 'Live interpretation session started for a bridged call.'),
    ('INTERPRETATION_SESSION_ENDED', 'clinical_ops', 'Live interpretation session ended.'),
    ('INTERPRETATION_PREFERENCES_UPDATED', 'clinical_ops', 'Interpretation language or preference settings were updated.'),
    ('INTERPRETATION_SEGMENT_PROCESSED', 'clinical_ops', 'An interpretation audio/text segment was processed.')
ON CONFLICT (event_type) DO NOTHING;
