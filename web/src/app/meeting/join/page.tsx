"use client";

import { Room, RoomEvent, Track } from "livekit-client";
import { useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useRef, useState } from "react";

function MeetingJoinInner() {
  const params = useSearchParams();
  const server = params.get("server") ?? "";
  const token = params.get("token") ?? "";
  const roomName = params.get("room") ?? "";
  const [status, setStatus] = useState("Connecting…");
  const [error, setError] = useState<string | null>(null);
  const roomRef = useRef<Room | null>(null);
  const localVideoRef = useRef<HTMLVideoElement>(null);
  const remoteContainerRef = useRef<HTMLDivElement>(null);

  const attachTrack = useCallback((track: Track, participantIdentity: string) => {
    if (track.kind !== Track.Kind.Video) {
      return;
    }
    const el = track.attach() as HTMLVideoElement;
    el.autoplay = true;
    el.playsInline = true;
    const room = roomRef.current;
    if (room && participantIdentity === room.localParticipant.identity) {
      if (localVideoRef.current) {
        localVideoRef.current.replaceWith(el);
        el.id = "local-video";
        localVideoRef.current = el;
      }
      return;
    }
    el.className = "h-full w-full rounded-xl bg-slate-900 object-cover";
    const wrap = document.createElement("div");
    wrap.className = "aspect-video overflow-hidden rounded-xl bg-slate-900";
    wrap.appendChild(el);
    remoteContainerRef.current?.appendChild(wrap);
  }, []);

  useEffect(() => {
    if (!server || !token) {
      setError("Missing LiveKit server URL or access token.");
      setStatus("");
      return;
    }

    const room = new Room({ adaptiveStream: true, dynacast: true });
    roomRef.current = room;

    room.on(RoomEvent.TrackSubscribed, (track, _pub, participant) => {
      attachTrack(track, participant.identity);
    });
    room.on(RoomEvent.TrackPublished, (pub, participant) => {
      if (pub.track) {
        attachTrack(pub.track, participant.identity);
      }
    });
    room.on(RoomEvent.Disconnected, () => {
      setStatus("Disconnected");
    });

    let cancelled = false;
    (async () => {
      try {
        await room.connect(server, token);
        if (cancelled) {
          return;
        }
        setStatus("Connected");
        await room.localParticipant.setCameraEnabled(true);
        await room.localParticipant.setMicrophoneEnabled(true);
        room.localParticipant.trackPublications.forEach((pub) => {
          if (pub.track) {
            attachTrack(pub.track, room.localParticipant.identity);
          }
        });
        if (roomName) {
          document.title = `Visit — ${roomName}`;
        }
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : "Could not join the visit.");
          setStatus("");
        }
      }
    })();

    return () => {
      cancelled = true;
      room.disconnect();
      roomRef.current = null;
    };
  }, [server, token, roomName, attachTrack]);

  return (
    <main className="mx-auto flex min-h-screen max-w-4xl flex-col gap-6 px-4 py-10">
      <div>
        <h1 className="text-2xl font-black text-slate-950">Zorba Health video visit</h1>
        {roomName ? (
          <p className="mt-1 text-sm font-semibold text-slate-500">Room: {roomName}</p>
        ) : null}
        {status ? (
          <p className="mt-2 text-sm font-semibold text-slate-600">{status}</p>
        ) : null}
        {error ? (
          <p className="mt-2 text-sm font-semibold text-red-600">{error}</p>
        ) : null}
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <div className="aspect-video overflow-hidden rounded-xl bg-slate-900">
          <video
            ref={localVideoRef}
            id="local-video"
            className="h-full w-full object-cover"
            autoPlay
            playsInline
            muted
          />
          <p className="pointer-events-none -mt-8 relative z-10 px-3 text-xs font-bold text-white">
            You
          </p>
        </div>
        <div ref={remoteContainerRef} className="grid gap-4" />
      </div>
    </main>
  );
}

export default function MeetingJoinPage() {
  return (
    <Suspense
      fallback={
        <main className="mx-auto max-w-4xl px-4 py-10 text-sm font-semibold text-slate-600">
          Loading…
        </main>
      }
    >
      <MeetingJoinInner />
    </Suspense>
  );
}
