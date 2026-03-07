# Email verification flow (Twilio + notification_service)

This document describes how a user is verified by email when they start registration: patient-service stores pending data and publishes a message; notification_service consumes `patient.event.chached` and sends a verification email via Twilio (SendGrid); the user completes registration by using the link.

---

## 1. End-to-end flow

1. **Client** calls `POST /api/v1/auth/patient/register` with email, phone, password, full name, date of birth.
2. **API Gateway** calls patient-service gRPC `StartRegistration`.
3. **Patient-service** validates input, generates a verification token, stores pending registration in **Redis** (TTL e.g. 15 min), then publishes to **RabbitMQ** with routing key `patient.event.chached` and does **not** create a user or patient yet.
4. **Notification-service** consumes messages with routing key `patient.event.chached`, reads token and email from the payload, and sends a **verification email** via **Twilio (SendGrid)** containing a link.
5. User clicks the link (e.g. frontend `/register/patient/verify?token=...`), which calls `POST /api/v1/auth/patient/register/verify` with `{"token":"..."}`.
6. **API Gateway** calls patient-service gRPC `VerifyEmail(token)`. **Patient-service** loads pending data from Redis, creates the user in auth-service, creates the patient, deletes the pending record, and publishes `patient.event.registered`. Registration is complete.

---

## 2. Message: patient.event.chached

- **Exchange:** `patient` (topic).
- **Routing key:** `patient.event.chached` (constant: `contracts.PatientEventChached`).
- **Body:** JSON of `contracts.AmqpMessage`:
  - `ownerId`: verification token (use as `token` in the verify link).
  - `data`: JSON bytes of `events.PatientEventData` with `register_request` set; `register_request` is `PendingRegistrationData` (email, phone_number, full_name, date_of_birth; no password).

Notification-service must: unmarshal body to AmqpMessage, take token from `ownerId`, unmarshal `data` to PatientEventData, and use `payload.RegisterRequest.Email` (and optionally FullName) to send the email with link `VERIFY_BASE_URL?token=<token>`.

---

## 3. Notification-service responsibilities

- **Connect** to RabbitMQ (env `RABBITMQ_URI`).
- **Declare** a queue (e.g. `notify_patient_pending_verification`), **bind** it to exchange `patient` with routing key `patient.event.chached`.
- **Consume** messages; for each:
  - Parse AmqpMessage then PatientEventData.
  - If `RegisterRequest` is nil, log and ack (or nack depending on policy).
  - Build verification URL: `VERIFY_BASE_URL + "?token=" + AmqpMessage.OwnerID` (or equivalent from env).
  - Send one email to `RegisterRequest.Email` via Twilio (e.g. SendGrid) with a clear "Verify your email" body and the link.
  - Ack the message on success; nack/requeue or dead-letter on failure as desired.
- **Env:** `RABBITMQ_URI`, `VERIFY_BASE_URL` (or `FRONTEND_VERIFY_URL`), and Twilio/SendGrid credentials (e.g. `SENDGRID_API_KEY`; or Twilio auth token and from address).

---

## 4. Twilio (SendGrid) email

- Use **Twilio SendGrid** (or Twilio's email API) to send transactional email.
- **From:** a verified sender (env e.g. `EMAIL_FROM`).
- **To:** `RegisterRequest.Email`.
- **Subject / body:** e.g. "Verify your email for ZorbaHealth" and a single prominent link: `VERIFY_BASE_URL?token=<token>`.
- Do not put the raw token in logs; log only that an email was sent (and optionally a redacted or hashed identifier).

---

## 5. Verification link and API

- **Link in email:** `VERIFY_BASE_URL?token=<token>` (e.g. `https://yourapp.com/register/patient/verify?token=...`).
- **Frontend:** Page at that path reads `token` from the query string and calls `POST /api/v1/auth/patient/register/verify` with body `{"token":"<token>"}`. On success, redirect to login or show success.
- **API:** Existing api-gateway handler and patient-service `VerifyEmail` gRPC complete user and patient creation and return `patient_id` and `user_id`.

---

## 6. Patient-service (reference)

- **StartRegistration:** Implemented in grpc_handler: calls `StartRegistrationWithVerification`, then `PublishPatientChached` with `(registerReq, token)`.
- **VerifyEmail:** Loads from Redis, creates user (auth-service) and patient, publishes `patient.event.registered`, returns ids.

---

## 7. Queue and binding (notification-service)

- Notification-service **declares its own queue** and binds it to exchange `patient` with routing key `patient.event.chached`. No change to shared messaging is required if notification-service creates the queue on startup.
- Alternatively, a dedicated queue for this routing key can be declared in shared messaging and consumed by notification-service; both approaches are valid.

---

## 8. Environment summary

| Service              | Variable         | Purpose                                                   |
| -------------------- | ---------------- | --------------------------------------------------------- |
| notification-service | RABBITMQ_URI     | RabbitMQ connection.                                      |
| notification-service | VERIFY_BASE_URL  | Base URL for the verify link (e.g. frontend verify page). |
| notification-service | SENDGRID_API_KEY | Twilio SendGrid API key for sending email.                |
| notification-service | EMAIL_FROM       | Verified sender address.                                  |
| patient-service      | (existing)       | Redis, RabbitMQ, auth-service; no change for this flow.   |

---

## 9. Implementation work items (plan)

- Add shared constant for notification queue (optional if notification-service declares its own queue).
- Create notification-service: main, RabbitMQ connect + declare queue + bind `patient.event.chached`, consumer handler, Twilio/SendGrid sender adapter, env wiring.
- Add notification-service env vars to K8s/ConfigMap/Secrets (RABBITMQ_URI, VERIFY_BASE_URL, SENDGRID_API_KEY, EMAIL_FROM).
