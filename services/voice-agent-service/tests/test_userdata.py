import asyncio

import pytest

from userdata import SessionUserData


@pytest.mark.asyncio
async def test_cancel_background_verification_cancels_task() -> None:
    ud = SessionUserData(verification_state="existing_patient_otp_pending")

    async def _block() -> None:
        await asyncio.sleep(3600)

    task = asyncio.create_task(_block())
    ud.otp_collection_task = task
    ud.cancel_background_verification()

    assert ud.verification_state == "call_ended"
    assert ud.otp_collection_task is None
    await asyncio.sleep(0)
    assert task.cancelled()


def test_cancel_background_verification_noop_when_not_polling() -> None:
    ud = SessionUserData(verification_state="anonymous")
    ud.cancel_background_verification()
    assert ud.verification_state == "anonymous"
    assert ud.otp_collection_task is None


def test_cancel_background_verification_preserves_verified_state() -> None:
    ud = SessionUserData(verification_state="verified_patient")
    ud.cancel_background_verification()
    assert ud.verification_state == "verified_patient"
