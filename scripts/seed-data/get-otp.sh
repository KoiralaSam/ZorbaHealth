#!/usr/bin/env bash
# Dev helper: fetch the pending-registration OTP and email-verification token
# for a phone number from the in-cluster Redis, since the local stack has no
# real SMS/email delivery. Use during registration-flow verification:
#
#   ./scripts/seed-data/get-otp.sh "+1 (555) 010-2233"
#
# Prints the JSON entry {"token": "...", "code": "..."} written by
# patient-service. `code` is the SMS OTP; `token` is the email-verify token
# accepted by POST /api/v1/auth/patient/register/verify.

set -euo pipefail

if [ $# -ne 1 ]; then
  echo "usage: $0 <phone-number>" >&2
  exit 2
fi

# Keys use canonical digits-only phone form (shared/auth CanonicalPhoneDigits).
digits="$(printf '%s' "$1" | tr -cd '0-9')"
if [ -z "$digits" ]; then
  echo "error: no digits in phone number '$1'" >&2
  exit 2
fi

entry="$(kubectl -n dev exec deploy/redis -- redis-cli --no-raw GET "otp:${digits}" 2>/dev/null | sed -e 's/^"//' -e 's/"$//' -e 's/\\"/"/g')"
if [ -z "$entry" ] || [ "$entry" = "(nil)" ]; then
  echo "No pending OTP for ${digits}. Start registration first (it expires with the pending-registration TTL)." >&2
  exit 1
fi
echo "$entry"
