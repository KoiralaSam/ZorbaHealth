# Sample Requests

These examples are intended for local development and documentation walkthroughs.

## Patient registration

```bash
curl -X POST http://localhost:8081/api/v1/auth/patient/register \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Demo",
    "last_name": "Patient",
    "email": "demo@example.com",
    "password": "replace-me"
  }'
```

## Patient login

```bash
curl -X POST http://localhost:8081/api/v1/auth/patient/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "demo@example.com",
    "password": "replace-me"
  }'
```

## Registration verification

```bash
curl -X POST http://localhost:8081/api/v1/auth/patient/register/verify \
  -H "Content-Type: application/json" \
  -d '{
    "email": "demo@example.com"
  }'
```

## Registration OTP verification

```bash
curl -X POST http://localhost:8081/api/v1/auth/patient/register/verify-otp \
  -H "Content-Type: application/json" \
  -d '{
    "email": "demo@example.com",
    "otp": "123456"
  }'
```

## gRPC exploration

Use `grpcurl` against service ports forwarded by Tilt.

Example pattern:

```bash
grpcurl -plaintext localhost:50054 list
```

As additional public APIs and gRPC services stabilize, expand this directory with request collections and sample payloads per service.
