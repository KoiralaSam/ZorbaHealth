from auth import SessionAuth


def test_mints_and_verifies_provisional_patient_token() -> None:
    auth = SessionAuth("test-secret", token_ttl_seconds=60)

    token = auth.mint_provisional_token("room-123")
    claims = auth.verify_patient_token(token)

    assert claims.patient_id == "session:room-123"
    assert claims.session_id == "room-123"
    assert claims.scopes == ["location:read"]
