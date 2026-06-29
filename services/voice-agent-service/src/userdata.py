"""
userdata.py – per-session state carried through the LiveKit Agents job.

SessionUserData is stored in AgentSession.userdata and mutated as the caller
progresses through identity verification. MCP remains the backend policy
boundary for PHI access.
"""

import asyncio
from dataclasses import dataclass, field
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from auth import SessionAuth
    from tools.mcp_client import MCPClient


@dataclass
class SessionUserData:
    verification_state: str = "anonymous"

    # Stable identifiers set at provision time.
    room_name: str = ""
    room_sid: str = ""
    session_id: str = ""
    caller_identity: str = ""
    caller_phone: str = ""
    language: str = "en"
    patient_id_hint: str = ""
    interpreter_mode: bool = False
    staff_identity: str = ""

    # Tokens: provisional until identity is verified, then upgraded.
    provisional_token: str = ""
    patient_token: str | None = None

    # Set after OTP verification.
    verified_patient_id: str | None = None

    # Tracks the active verification path so tools know which OTP to submit.
    verification_mode: str = ""  # "existing" | ""
    verification_correlation_id: str = ""
    dtmf_otp_buffer: str = ""
    verification_attempts: int = 0

    # Health context loaded after verification; summarised by Go.
    health_context: str = ""

    # Safety / escalation state for the current voice session.
    escalation_triggered: bool = False
    escalation_reason: str = ""
    escalation_guidance: str = ""
    transfer_requested: bool = False
    transfer_target: str = ""
    last_user_transcript: str = ""
    last_user_transcript_language: str = "en"
    alert_phone_numbers: list[str] | None = None

    # HTTP MCP client (set once per job in agent entrypoint).
    mcp_client: "MCPClient | None" = None
    session_auth: "SessionAuth | None" = None

    # Background poll for inbound-SMS OTP completion (see verification.wait_for_sms_verification).
    otp_collection_task: asyncio.Task | None = field(default=None, repr=False)

    @property
    def active_token(self) -> str:
        """Return the best available token for tool API calls."""
        return self.patient_token or self.provisional_token

    @property
    def is_verified(self) -> bool:
        """Return True if the caller has passed OTP verification."""
        return self.verified_patient_id is not None and self.patient_token is not None

    def upgrade(self, patient_id: str, patient_token: str) -> None:
        """Record a successful verification and store the upgraded token."""
        self.verified_patient_id = patient_id
        self.patient_token = patient_token
        self.verification_state = "verified_patient"
        self.verification_mode = ""

    def cancel_background_verification(self) -> None:
        """Stop SMS OTP polling and other verification background work for this call."""
        if self.verification_state == "existing_patient_otp_pending":
            self.verification_state = "call_ended"
        task = self.otp_collection_task
        if task is not None and not task.done():
            task.cancel()
        self.otp_collection_task = None
