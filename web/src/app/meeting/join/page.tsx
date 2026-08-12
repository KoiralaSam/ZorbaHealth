"use client";

import "@livekit/components-styles";
import {
  LiveKitRoom,
  RoomAudioRenderer,
  VideoConference,
} from "@livekit/components-react";
import { useSearchParams } from "next/navigation";
import { Suspense, useMemo, useState } from "react";

function MeetingJoinInner() {
  const params = useSearchParams();
  const server = (params.get("server") ?? "").trim();
  const token = (params.get("token") ?? "").trim();
  const roomName = (params.get("room") ?? "").trim();
  const [disconnected, setDisconnected] = useState(false);

  const missing = useMemo(() => !server || !token, [server, token]);

  if (missing) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-slate-950 px-4">
        <div className="max-w-md rounded-2xl border border-white/10 bg-slate-900/80 p-6 text-center shadow-2xl">
          <h1 className="text-xl font-semibold text-white">Missing join details</h1>
          <p className="mt-2 text-sm text-slate-300">
            This visit link needs a LiveKit server URL and access token. Open the link from your
            Zorba Health meeting email or dashboard.
          </p>
        </div>
      </main>
    );
  }

  if (disconnected) {
    return (
      <main className="flex min-h-screen items-center justify-center bg-slate-950 px-4">
        <div className="max-w-md rounded-2xl border border-white/10 bg-slate-900/80 p-6 text-center shadow-2xl">
          <h1 className="text-xl font-semibold text-white">Visit ended</h1>
          <p className="mt-2 text-sm text-slate-300">
            You left the LiveKit room{roomName ? ` (${roomName})` : ""}. You can close this tab.
          </p>
          <button
            type="button"
            className="mt-5 rounded-full bg-indigo-500 px-5 py-2 text-sm font-semibold text-white hover:bg-indigo-400"
            onClick={() => {
              setDisconnected(false);
            }}
          >
            Rejoin
          </button>
        </div>
      </main>
    );
  }

  return (
    <main className="h-[100dvh] w-full overflow-hidden bg-slate-950 text-white">
      <div className="absolute left-4 top-4 z-20 rounded-full bg-black/50 px-3 py-1.5 text-xs font-semibold tracking-wide text-white/90 backdrop-blur">
        Zorba Health{roomName ? ` · ${roomName}` : ""}
      </div>
      <LiveKitRoom
        token={token}
        serverUrl={server}
        connect
        video
        audio
        data-lk-theme="default"
        className="h-full w-full"
        onDisconnected={() => setDisconnected(true)}
        onError={(err) => {
          console.error("LiveKit meeting error", err);
        }}
      >
        {/* VideoConference focuses ScreenShare over Camera when present. */}
        <VideoConference />
        <RoomAudioRenderer />
      </LiveKitRoom>
    </main>
  );
}

export default function MeetingJoinPage() {
  return (
    <Suspense
      fallback={
        <main className="flex min-h-screen items-center justify-center bg-slate-950 text-sm font-semibold text-slate-300">
          Connecting to visit…
        </main>
      }
    >
      <MeetingJoinInner />
    </Suspense>
  );
}
