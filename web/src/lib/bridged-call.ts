import type { PatientCallSummary } from "../contracts";

/** Call row the backend still marks as in progress. */
export function findActivePatientCall(
  calls: PatientCallSummary[],
): PatientCallSummary | null {
  const active = calls.find(
    (call) =>
      call.status?.toLowerCase() === "active" &&
      Boolean(call.livekit_room_id?.trim()),
  );
  if (active) {
    return active;
  }
  return (
    calls.find(
      (call) => Boolean(call.livekit_room_id?.trim()) && !call.ended_at,
    ) ?? null
  );
}

/** Session id for bridge APIs: live voice WS session first, then active call row. */
export function resolveBridgedCallSessionId(
  liveVoiceSessionId: string | null | undefined,
  calls: PatientCallSummary[],
): string | null {
  const fromVoice = liveVoiceSessionId?.trim();
  if (fromVoice) {
    return fromVoice;
  }
  const room = findActivePatientCall(calls)?.livekit_room_id?.trim();
  return room || null;
}

export function isCallInProgress(
  liveVoiceSessionId: string | null | undefined,
  calls: PatientCallSummary[],
): boolean {
  return resolveBridgedCallSessionId(liveVoiceSessionId, calls) != null;
}
