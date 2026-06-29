from auth import SessionAuth
from safety import evaluate_text


def test_mints_and_verifies_provisional_patient_token() -> None:
    auth = SessionAuth("test-secret", token_ttl_seconds=60)

    token = auth.mint_provisional_token("room-123", "15551234567")
    claims = auth.verify_patient_token(token)

    assert claims.patient_id == "session:room-123"
    assert claims.session_id == "room-123"
    assert claims.scopes == ["location:read"]


def test_provisional_token_includes_caller_phone_in_payload() -> None:
    import jwt

    auth = SessionAuth("test-secret", token_ttl_seconds=60)
    token = auth.mint_provisional_token("room-123", "+1 (555) 123-4567")
    payload = jwt.decode(token, options={"verify_signature": False})
    assert payload.get("callerPhone") == "15551234567"


def test_mints_verified_patient_token_with_records_scope() -> None:
    auth = SessionAuth("test-secret", token_ttl_seconds=60)

    token = auth.mint_patient_token(
        patient_id="patient-123",
        session_id="room-123",
        scopes=["location:read", "records:read"],
    )
    claims = auth.verify_patient_token(token)

    assert claims.patient_id == "patient-123"
    assert claims.session_id == "room-123"
    assert claims.scopes == ["location:read", "records:read"]


def test_emergency_workflow_flags_high_risk_transcript() -> None:
    decision = evaluate_text("I have chest pain and shortness of breath")

    assert decision.should_escalate is True
    assert decision.reason
    assert "9-1-1" in decision.guidance
