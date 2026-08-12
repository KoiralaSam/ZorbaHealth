export type MeetingJoinPayload = {
  server: string;
  token: string;
  roomName?: string;
  fallbackURL?: string;
};

export type MeetingJoinFields = {
  join_url?: string;
  livekit_server_url?: string;
  participant_token?: string;
  livekit_room_name?: string;
  status?: string;
};

export function parseMeetingJoinURL(
  joinURL: string,
): { server: string; room: string; token: string } | null {
  const raw = (joinURL || "").trim();
  if (!raw) return null;
  try {
    const url = new URL(raw);
    const server = (url.searchParams.get("server") || "").trim();
    const room = (url.searchParams.get("room") || "").trim();
    const token = (url.searchParams.get("token") || "").trim();
    if (server && token) {
      return { server, room, token };
    }
    if (raw.startsWith("ws://") || raw.startsWith("wss://")) {
      const wsToken = (url.searchParams.get("token") || "").trim();
      if (wsToken) {
        return {
          server: `${url.protocol}//${url.host}${url.pathname}`.replace(/\/$/, ""),
          room,
          token: wsToken,
        };
      }
    }
  } catch {
    return null;
  }
  return null;
}

/** Prefer API token fields; fall back to parsing join_url query params. */
export function resolveMeetingJoin(
  meeting: MeetingJoinFields,
): MeetingJoinPayload | null {
  const server = (meeting.livekit_server_url || "").trim();
  const token = (meeting.participant_token || "").trim();
  const roomName = (meeting.livekit_room_name || "").trim();
  const fallbackURL = (meeting.join_url || "").trim();

  if (server && token) {
    return {
      server,
      token,
      roomName: roomName || undefined,
      fallbackURL: fallbackURL || undefined,
    };
  }

  const parsed = parseMeetingJoinURL(fallbackURL);
  if (!parsed) return null;
  return {
    server: parsed.server,
    token: parsed.token,
    roomName: parsed.room || roomName || undefined,
    fallbackURL: fallbackURL || undefined,
  };
}

export function meetingCanJoin(meeting: MeetingJoinFields): boolean {
  return resolveMeetingJoin(meeting) != null;
}

export function isHttpURL(value?: string): boolean {
  if (!value) return false;
  return value.startsWith("http://") || value.startsWith("https://");
}
