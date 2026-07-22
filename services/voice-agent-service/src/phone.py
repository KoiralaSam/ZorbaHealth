"""Canonical phone helpers aligned with shared/auth phone.go."""

from __future__ import annotations


def normalize_phone_digits(phone: str) -> str:
    return "".join(c for c in phone if c.isdigit())


def canonical_phone_digits(phone: str) -> str:
    """Digits-only E.164 without '+'. NANP is always 11 digits with leading 1."""
    digits = normalize_phone_digits(phone)
    if not digits:
        return ""
    if len(digits) == 10:
        return "1" + digits
    if len(digits) == 11 and digits.startswith("1"):
        return digits
    return digits
