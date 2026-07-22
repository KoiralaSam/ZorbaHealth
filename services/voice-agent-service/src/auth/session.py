from __future__ import annotations

import time
from dataclasses import dataclass, field

import jwt


@dataclass(frozen=True)
class SessionClaims:
    patient_id: str
    session_id: str
    scopes: list[str] = field(default_factory=list)


class SessionAuth:
    def __init__(
        self,
        patient_jwt_secret: str,
        issuer: str = "zorba-voice-agent-service",
        token_ttl_seconds: int = 7200,
    ) -> None:
        if not patient_jwt_secret:
            raise ValueError("patient_jwt_secret is required")
        self._patient_jwt_secret = patient_jwt_secret
        self._issuer = issuer
        self._token_ttl_seconds = token_ttl_seconds

    def mint_provisional_token(self, session_id: str, caller_phone: str = "") -> str:
        return self.mint_patient_token(
            patient_id=f"session:{session_id}",
            session_id=session_id,
            scopes=["location:read"],
            caller_phone=caller_phone,
        )

    def mint_patient_token(
        self,
        patient_id: str,
        session_id: str,
        scopes: list[str],
        caller_phone: str = "",
    ) -> str:
        now = int(time.time())
        payload = {
            "actorType": "patient",
            "patientID": patient_id,
            "sessionID": session_id,
            "scopes": scopes,
            "iss": self._issuer,
            "iat": now,
            "exp": now + self._token_ttl_seconds,
        }
        digits = "".join(c for c in caller_phone if c.isdigit())
        if digits:
            # Keep JWT callerPhone in the same canonical form as users/patients storage.
            from phone import canonical_phone_digits

            payload["callerPhone"] = canonical_phone_digits(digits) or digits
        return jwt.encode(payload, self._patient_jwt_secret, algorithm="HS256")

    def verify_patient_token(self, token: str) -> SessionClaims:
        payload = jwt.decode(token, self._patient_jwt_secret, algorithms=["HS256"])
        if payload.get("actorType") != "patient":
            raise ValueError("token is not a patient token")
        patient_id = str(payload.get("patientID") or "")
        session_id = str(payload.get("sessionID") or "")
        if not patient_id or not session_id:
            raise ValueError("token is missing patient/session claims")
        scopes = payload.get("scopes") or []
        if not isinstance(scopes, list):
            raise ValueError("token scopes must be a list")
        return SessionClaims(patient_id=patient_id, session_id=session_id, scopes=scopes)
