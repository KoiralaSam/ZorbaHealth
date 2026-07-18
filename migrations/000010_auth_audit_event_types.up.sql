INSERT INTO audit.audit_event_types (event_type, retention_class, description) VALUES
    ('PATIENT_SESSION_REFRESH', 'security', 'Patient session refreshed via refresh token.'),
    ('STAFF_LOGIN', 'security', 'Hospital staff login succeeded.'),
    ('STAFF_LOGOUT', 'security', 'Hospital staff logout completed.'),
    ('STAFF_SESSION_REFRESH', 'security', 'Staff session refreshed via refresh token.'),
    ('SESSION_REUSE_DETECTED', 'security', 'Rotated refresh token reuse detected; session family revoked.')
ON CONFLICT (event_type) DO NOTHING;
