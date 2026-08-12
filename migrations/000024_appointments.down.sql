DROP TABLE IF EXISTS appointments;
DROP TABLE IF EXISTS staff_availability_exceptions;
DROP TABLE IF EXISTS staff_availability_rules;

DELETE FROM audit.audit_event_types WHERE event_type IN (
    'APPOINTMENT_BOOKED',
    'APPOINTMENT_RESCHEDULED',
    'APPOINTMENT_CANCELLED',
    'APPOINTMENT_BOOK_DENIED',
    'AVAILABILITY_UPDATED'
);
