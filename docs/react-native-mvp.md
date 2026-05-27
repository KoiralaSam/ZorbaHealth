# React Native MVP

This document defines the first achievable React Native companion app for Zorba Health.

## Goal

The mobile app should extend the phone-first experience, not replace it.

The MVP is intentionally narrow:

- secure patient login
- consent center
- one-tap call entry into the existing voice flow
- recent call summaries
- optional location sharing entry point for emergency sessions

In-app voice is explicitly deferred until the call state machine and agent lifecycle are more mature.

## Recommended stack

- Expo
- React Native
- TypeScript
- the same HTTP API surface already used by the Next.js web app

## API contract

The app should use the same gateway endpoints as the patient web portal:

- `POST /api/v1/auth/patient/login`
- `GET /api/v1/patient/profile`
- `GET /api/v1/patient/consents`
- `POST /api/v1/patient/consents`
- `DELETE /api/v1/patient/consents`
- `GET /api/v1/patient/calls`
- `POST /api/v1/patient/records/answer`

This keeps web and mobile behavior aligned and avoids duplicate backend logic.

## Screens

### 1. Login

- collect phone number and password
- store the returned token in secure device storage
- redirect to the patient home screen on success

### 2. Home

- show profile name and support number
- show a `Call Zorba` action using the existing phone-based voice path
- show a compact recent-activity panel

### 3. Consent center

- list every modeled consent type
- grant or revoke each consent using the gateway
- explain why each consent exists in plain language

### 4. Call summaries

- list recent call summaries from `GET /api/v1/patient/calls`
- do not expose raw transcripts by default

### 5. Emergency location sharing

- only shown when the patient explicitly opts in (`LOCATION_ACCESS` consent)
- maintain a WebSocket to **location-service** (`GET /ws/location?token=...`), not the API gateway
- when `start_location` arrives (after `call.started`), begin `watchPosition` and send `location_update` JSON with the voice `sessionID`
- stop GPS on `stop_location` or when the socket closes

## Security requirements

- store tokens in secure storage, not plain async storage
- do not log PHI
- do not cache raw health-record payloads longer than necessary
- keep network access limited to the API gateway

## Sequencing

1. Reuse the existing patient portal API surface.
2. Build read-only screens first.
3. Add consent mutation flows.
4. Add push or callback notifications only after notification templates and channel work are ready.

## Deferred features

- in-app LiveKit or WebRTC calling
- full appointment scheduling
- full offline patient record access
- patient analytics or dashboards based on operator data
