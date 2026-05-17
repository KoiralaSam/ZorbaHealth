"""
userdata.py – per-session state carried through the LiveKit Agents job.

SessionUserData is stored in AgentSession.userdata and mutated as the caller
progresses through identity verification. MCP remains the backend policy
boundary for PHI access.
"""

from dataclasses import dataclass
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from tools.mcp_client import MCPClient


@dataclass
class SessionUserData:
    # Stable identifiers set at provision time.
    room_name: str = ""
    room_sid: str = ""
    session_id: str = ""
    caller_phone: str = ""
    language: str = "en"
    patient_id_hint: str = ""

    # Tokens: provisional until identity is verified, then upgraded.
    provisional_token: str = ""
    patient_token: str | None = None

    # Set after OTP verification or registration.
    verified_patient_id: str | None = None

    # In-flight registration token returned by start-registration.
    registration_token: str | None = None

    # Tracks the active verification path so tools know which OTP to submit.
    verification_mode: str = ""  # "existing" | "registration" | ""

    # Health context loaded after verification; summarised by Go.
    health_context: str = ""

    # HTTP MCP client (set once per job in agent entrypoint).
    mcp_client: "MCPClient | None" = None

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
        self.verification_mode = ""
        self.registration_token = None
