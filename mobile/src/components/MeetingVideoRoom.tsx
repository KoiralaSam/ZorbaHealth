import { Ionicons } from "@expo/vector-icons";
import {
  AudioSession,
  LiveKitRoom,
  VideoTrack,
  isTrackReference,
  useConnectionState,
  useLocalParticipant,
  useRoomContext,
  useTracks,
  type TrackReferenceOrPlaceholder,
} from "@livekit/react-native";
import {
  ScreenCapturePickerView,
} from "@livekit/react-native-webrtc";
import { ConnectionState, Track } from "livekit-client";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  findNodeHandle,
  Linking,
  Modal,
  NativeModules,
  Platform,
  Pressable,
  StatusBar,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import {
  isHttpURL,
  type MeetingJoinPayload,
} from "../lib/meetingJoin";

export type { MeetingJoinPayload, MeetingJoinFields } from "../lib/meetingJoin";
export {
  meetingCanJoin,
  parseMeetingJoinURL,
  resolveMeetingJoin,
} from "../lib/meetingJoin";

/** True when Expo Go / missing native WebRTC module left registerGlobals unset. */
let liveKitNativeReady = false;

export function markLiveKitNativeReady(ready: boolean) {
  liveKitNativeReady = ready;
}

export function isLiveKitNativeReady() {
  return liveKitNativeReady;
}

function shortRoomLabel(roomName?: string) {
  const raw = (roomName || "").trim();
  if (!raw) return "Video visit";
  const m = raw.match(/^meeting-([0-9a-f-]{8})/i);
  if (m) return `Visit · ${m[1]}`;
  if (raw.length > 28) return `${raw.slice(0, 26)}…`;
  return raw;
}

function formatJoinError(err: unknown, server: string): string {
  const raw = err instanceof Error ? err.message : "Unable to join the video visit.";
  const lower = raw.toLowerCase();
  if (
    lower.includes("signal connection") ||
    lower.includes("abort handler") ||
    lower.includes("websocket") ||
    lower.includes("failed to fetch") ||
    lower.includes("network request failed")
  ) {
    return (
      `${raw}\n\n` +
      `Cannot reach LiveKit at ${server}. ` +
      `Open GCP firewall tcp:7880, tcp:7881, udp:50000-50100 ` +
      `(Cloud Shell: deploy/tilt/open-livekit-firewall.sh), then retry Join.`
    );
  }
  return raw;
}

function isScreenShareTrack(track: TrackReferenceOrPlaceholder) {
  return track.source === Track.Source.ScreenShare;
}

function trackLabel(track: TrackReferenceOrPlaceholder) {
  const base = track.participant.isLocal
    ? "You"
    : track.participant.name || track.participant.identity || "Participant";
  return isScreenShareTrack(track) ? `${base} · Screen` : base;
}

/**
 * Prefer screen share over camera so a shared screen replaces the camera feed
 * for both the sharer and remote viewers.
 */
function pickPrimaryAndPip(tracks: TrackReferenceOrPlaceholder[]): {
  primary?: TrackReferenceOrPlaceholder;
  pip?: TrackReferenceOrPlaceholder;
} {
  const screenShares = tracks.filter(isScreenShareTrack);
  const cameras = tracks.filter((t) => t.source === Track.Source.Camera);

  const remoteScreen = screenShares.find((t) => !t.participant.isLocal);
  const localScreen = screenShares.find((t) => t.participant.isLocal);
  const remoteCamera = cameras.find((t) => !t.participant.isLocal);
  const localCamera = cameras.find((t) => t.participant.isLocal);

  const primary = remoteScreen ?? localScreen ?? remoteCamera ?? localCamera;
  if (!primary) return {};

  if (isScreenShareTrack(primary)) {
    // Screen is main stage; keep a camera in PiP when available.
    if (primary.participant.isLocal) {
      return { primary, pip: remoteCamera ?? localCamera };
    }
    return { primary, pip: localCamera ?? remoteCamera };
  }

  // Camera call: remote focus, local PiP.
  if (!primary.participant.isLocal && localCamera) {
    return { primary, pip: localCamera };
  }
  return { primary };
}

function ControlButton({
  icon,
  label,
  onPress,
  danger,
  muted,
  active,
}: {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  danger?: boolean;
  muted?: boolean;
  active?: boolean;
}) {
  return (
    <Pressable
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => [
        styles.controlBtn,
        danger ? styles.controlLeave : null,
        !danger && muted ? styles.controlMuted : null,
        !danger && active ? styles.controlActive : null,
        pressed ? styles.controlPressed : null,
      ]}
    >
      <Ionicons name={icon} size={danger ? 26 : 22} color="#fff" />
    </Pressable>
  );
}

function ParticipantTile({
  trackRef,
  label,
  style,
  mirror,
  zOrder,
  objectFit = "cover",
}: {
  trackRef: TrackReferenceOrPlaceholder;
  label: string;
  style?: object;
  mirror?: boolean;
  zOrder?: number;
  objectFit?: "cover" | "contain";
}) {
  const hasVideo = isTrackReference(trackRef) && !!trackRef.publication?.track;
  return (
    <View style={[styles.tile, style]}>
      {hasVideo && isTrackReference(trackRef) ? (
        <VideoTrack
          trackRef={trackRef}
          style={styles.videoFill}
          objectFit={objectFit}
          mirror={mirror}
          zOrder={zOrder}
        />
      ) : (
        <View style={styles.tilePlaceholder}>
          <View style={styles.avatar}>
            <Text style={styles.avatarText}>
              {(label || "?").slice(0, 1).toUpperCase()}
            </Text>
          </View>
        </View>
      )}
      <View style={styles.tileLabelWrap}>
        <Text style={styles.tileLabel} numberOfLines={1}>
          {label}
        </Text>
      </View>
    </View>
  );
}

function ConferenceView({
  roomName,
  onLeave,
  fallbackURL,
  connectError,
}: {
  roomName?: string;
  onLeave: () => void;
  fallbackURL?: string;
  connectError: string;
}) {
  const insets = useSafeAreaInsets();
  const room = useRoomContext();
  const connectionState = useConnectionState();
  const { localParticipant, isMicrophoneEnabled, isCameraEnabled } =
    useLocalParticipant();
  const tracks = useTracks(
    [Track.Source.Camera, Track.Source.ScreenShare],
    { onlySubscribed: false },
  );
  const screenCaptureRef = useRef(null);
  const [screenShareBusy, setScreenShareBusy] = useState(false);

  const isScreenShareEnabled = useMemo(
    () =>
      tracks.some(
        (t) => t.participant.isLocal && t.source === Track.Source.ScreenShare,
      ),
    [tracks],
  );

  const { primary, pip } = useMemo(() => pickPrimaryAndPip(tracks), [tracks]);

  const connecting =
    connectionState === ConnectionState.Connecting ||
    connectionState === ConnectionState.Reconnecting;
  const connected = connectionState === ConnectionState.Connected;

  const toggleScreenShare = useCallback(async () => {
    if (screenShareBusy) return;
    setScreenShareBusy(true);
    try {
      const enable = !isScreenShareEnabled;
      if (enable && Platform.OS === "ios") {
        const reactTag = findNodeHandle(screenCaptureRef.current);
        if (reactTag) {
          await NativeModules.ScreenCapturePickerViewManager?.show?.(reactTag);
        }
      }
      await localParticipant.setScreenShareEnabled(enable);
    } catch (err) {
      console.warn("screen share toggle failed", err);
    } finally {
      setScreenShareBusy(false);
    }
  }, [isScreenShareEnabled, localParticipant, screenShareBusy]);

  return (
    <View style={styles.shell}>
      <StatusBar barStyle="light-content" backgroundColor="#0b0b0f" />

      <View style={[styles.topBar, { paddingTop: insets.top + 8 }]}>
        <View style={styles.brandPill}>
          <Text style={styles.brandText} numberOfLines={1}>
            Zorba Health · {shortRoomLabel(roomName)}
            {isScreenShareEnabled || (primary && isScreenShareTrack(primary))
              ? " · Sharing"
              : ""}
          </Text>
        </View>
        <Text style={styles.connectionMeta}>
          {connecting ? "Connecting…" : connected ? "Connected" : "Offline"}
        </Text>
      </View>

      <View style={styles.stage}>
        {primary ? (
          <ParticipantTile
            trackRef={primary}
            label={trackLabel(primary)}
            style={styles.focusTile}
            mirror={
              primary.participant.isLocal && !isScreenShareTrack(primary)
            }
            objectFit={isScreenShareTrack(primary) ? "contain" : "cover"}
            zOrder={0}
          />
        ) : (
          <View style={[styles.focusTile, styles.tilePlaceholder]}>
            <Text style={styles.waitingText}>
              {connecting
                ? "Connecting…"
                : connected
                  ? "Waiting for the other participant…"
                  : "Not connected"}
            </Text>
          </View>
        )}

        {pip ? (
          <View style={styles.pip}>
            <ParticipantTile
              trackRef={pip}
              label={trackLabel(pip)}
              style={styles.pipInner}
              mirror={pip.participant.isLocal && !isScreenShareTrack(pip)}
              objectFit={isScreenShareTrack(pip) ? "contain" : "cover"}
              zOrder={1}
            />
          </View>
        ) : null}

        {connectError ? (
          <View style={styles.errorBanner}>
            <Text style={styles.errorText}>{connectError}</Text>
            {isHttpURL(fallbackURL) ? (
              <Pressable onPress={() => Linking.openURL(fallbackURL as string)}>
                <Text style={styles.fallbackLink}>Open in browser</Text>
              </Pressable>
            ) : null}
          </View>
        ) : null}
      </View>

      <View style={[styles.controlBar, { paddingBottom: Math.max(insets.bottom, 18) }]}>
        <ControlButton
          icon={isMicrophoneEnabled ? "mic" : "mic-off"}
          label={isMicrophoneEnabled ? "Mute" : "Unmute"}
          muted={!isMicrophoneEnabled}
          onPress={() => {
            void localParticipant.setMicrophoneEnabled(!isMicrophoneEnabled);
          }}
        />
        <ControlButton
          icon={isCameraEnabled ? "videocam" : "videocam-off"}
          label={isCameraEnabled ? "Camera off" : "Camera on"}
          muted={!isCameraEnabled}
          onPress={() => {
            void localParticipant.setCameraEnabled(!isCameraEnabled);
          }}
        />
        <ControlButton
          icon="desktop-outline"
          label={isScreenShareEnabled ? "Stop sharing" : "Share screen"}
          active={isScreenShareEnabled}
          onPress={() => {
            void toggleScreenShare();
          }}
        />
        <ControlButton
          icon="call"
          label="Leave"
          danger
          onPress={() => {
            void room.disconnect();
            onLeave();
          }}
        />
      </View>

      {Platform.OS === "ios" ? (
        <View style={styles.screenCaptureHost} pointerEvents="none">
          <ScreenCapturePickerView ref={screenCaptureRef} />
        </View>
      ) : null}
    </View>
  );
}

function EndedView({
  roomName,
  onRejoin,
  onClose,
}: {
  roomName?: string;
  onRejoin: () => void;
  onClose: () => void;
}) {
  const insets = useSafeAreaInsets();
  return (
    <View style={[styles.centered, { paddingTop: insets.top, paddingBottom: insets.bottom }]}>
      <StatusBar barStyle="light-content" backgroundColor="#0b0b0f" />
      <View style={styles.card}>
        <Text style={styles.cardTitle}>Visit ended</Text>
        <Text style={styles.cardBody}>
          You left the LiveKit room{roomName ? ` (${shortRoomLabel(roomName)})` : ""}.
        </Text>
        <Pressable style={styles.primaryBtn} onPress={onRejoin}>
          <Text style={styles.primaryBtnText}>Rejoin</Text>
        </Pressable>
        <Pressable style={styles.secondaryBtn} onPress={onClose}>
          <Text style={styles.secondaryBtnText}>Back to meetings</Text>
        </Pressable>
      </View>
    </View>
  );
}

function FullscreenGate({
  children,
  onClose,
}: {
  children: React.ReactNode;
  onClose: () => void;
}) {
  return (
    <Modal
      visible
      animationType="slide"
      presentationStyle="fullScreen"
      statusBarTranslucent
      onRequestClose={onClose}
    >
      {children}
    </Modal>
  );
}

export function MeetingVideoRoom({
  join,
  onClose,
}: {
  join: MeetingJoinPayload;
  onClose: () => void;
}) {
  const [disconnected, setDisconnected] = useState(false);
  const [connectError, setConnectError] = useState("");
  const [sessionKey, setSessionKey] = useState(0);

  useEffect(() => {
    (async () => {
      try {
        await AudioSession.startAudioSession();
      } catch {
        // Native audio session may be unavailable in Expo Go.
      }
    })();
    return () => {
      void AudioSession.stopAudioSession().catch(() => undefined);
    };
  }, []);

  if (!liveKitNativeReady) {
    return (
      <FullscreenGate onClose={onClose}>
        <View style={styles.centered}>
          <View style={styles.card}>
            <Text style={styles.cardTitle}>Development build required</Text>
            <Text style={styles.cardBody}>
              LiveKit WebRTC is not available in Expo Go. Build a dev client with
              eas build --profile development --platform android.
            </Text>
            <Pressable style={styles.secondaryBtn} onPress={onClose}>
              <Text style={styles.secondaryBtnText}>Back</Text>
            </Pressable>
          </View>
        </View>
      </FullscreenGate>
    );
  }

  const server = join.server.trim();
  const token = join.token.trim();
  if (!server || !token) {
    return (
      <FullscreenGate onClose={onClose}>
        <View style={styles.centered}>
          <View style={styles.card}>
            <Text style={styles.cardTitle}>Missing join details</Text>
            <Text style={styles.cardBody}>
              This visit link needs a LiveKit server URL and access token.
            </Text>
            <Pressable style={styles.secondaryBtn} onPress={onClose}>
              <Text style={styles.secondaryBtnText}>Back</Text>
            </Pressable>
          </View>
        </View>
      </FullscreenGate>
    );
  }

  if (disconnected) {
    return (
      <FullscreenGate onClose={onClose}>
        <EndedView
          roomName={join.roomName}
          onClose={onClose}
          onRejoin={() => {
            setConnectError("");
            setDisconnected(false);
            setSessionKey((k) => k + 1);
          }}
        />
      </FullscreenGate>
    );
  }

  return (
    <FullscreenGate onClose={onClose}>
      <View style={styles.shell}>
        <LiveKitRoom
          key={sessionKey}
          serverUrl={server}
          token={token}
          connect
          audio
          video
          options={{
            adaptiveStream: { pixelDensity: "screen" },
            dynacast: true,
          }}
          onDisconnected={() => setDisconnected(true)}
          onError={(err) => setConnectError(formatJoinError(err, server))}
        >
          <ConferenceView
            roomName={join.roomName}
            fallbackURL={join.fallbackURL}
            connectError={connectError}
            onLeave={() => {
              setDisconnected(true);
            }}
          />
        </LiveKitRoom>
      </View>
    </FullscreenGate>
  );
}

const styles = StyleSheet.create({
  shell: {
    flex: 1,
    width: "100%",
    height: "100%",
    backgroundColor: "#0b0b0f",
  },
  topBar: {
    zIndex: 20,
    paddingHorizontal: 16,
    paddingBottom: 8,
    gap: 6,
    backgroundColor: "#0b0b0f",
  },
  brandPill: {
    alignSelf: "flex-start",
    maxWidth: "92%",
    borderRadius: 999,
    backgroundColor: "rgba(255,255,255,0.08)",
    paddingHorizontal: 12,
    paddingVertical: 7,
  },
  brandText: {
    color: "rgba(255,255,255,0.92)",
    fontSize: 12,
    fontWeight: "700",
    letterSpacing: 0.2,
  },
  connectionMeta: {
    color: "#64748b",
    fontSize: 11,
    fontWeight: "600",
  },
  stage: {
    flex: 1,
    backgroundColor: "#111118",
    position: "relative",
    overflow: "hidden",
  },
  focusTile: {
    ...StyleSheet.absoluteFillObject,
  },
  pip: {
    position: "absolute",
    right: 14,
    bottom: 14,
    width: 110,
    height: 156,
    borderRadius: 14,
    overflow: "hidden",
    borderWidth: 2,
    borderColor: "rgba(255,255,255,0.28)",
    backgroundColor: "#1a1a22",
    zIndex: 10,
  },
  pipInner: {
    flex: 1,
  },
  tile: {
    overflow: "hidden",
    backgroundColor: "#1a1a22",
  },
  videoFill: {
    width: "100%",
    height: "100%",
  },
  tilePlaceholder: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: "#1a1a22",
  },
  avatar: {
    width: 72,
    height: 72,
    borderRadius: 36,
    backgroundColor: "#3730a3",
    alignItems: "center",
    justifyContent: "center",
  },
  avatarText: {
    color: "#fff",
    fontSize: 28,
    fontWeight: "800",
  },
  waitingText: {
    color: "#94a3b8",
    fontSize: 15,
    fontWeight: "600",
    textAlign: "center",
    paddingHorizontal: 24,
  },
  tileLabelWrap: {
    position: "absolute",
    left: 10,
    bottom: 10,
    borderRadius: 8,
    backgroundColor: "rgba(0,0,0,0.55)",
    paddingHorizontal: 8,
    paddingVertical: 4,
  },
  tileLabel: {
    color: "#fff",
    fontSize: 12,
    fontWeight: "700",
    maxWidth: 160,
  },
  controlBar: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    flexWrap: "wrap",
    gap: 14,
    paddingTop: 16,
    paddingHorizontal: 12,
    backgroundColor: "#0b0b0f",
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: "rgba(255,255,255,0.08)",
  },
  controlBtn: {
    width: 52,
    height: 52,
    borderRadius: 26,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: "#2a2a33",
  },
  controlMuted: {
    backgroundColor: "#4b5563",
  },
  controlActive: {
    backgroundColor: "#4f46e5",
  },
  controlLeave: {
    backgroundColor: "#dc2626",
    width: 60,
    height: 60,
    borderRadius: 30,
  },
  controlPressed: {
    opacity: 0.85,
  },
  screenCaptureHost: {
    // Required in the tree for iOS RPSystemBroadcastPickerView; keep off-screen.
    position: "absolute",
    width: 1,
    height: 1,
    opacity: 0,
  },
  errorBanner: {
    position: "absolute",
    left: 12,
    right: 12,
    top: 12,
    borderRadius: 12,
    backgroundColor: "#450a0a",
    borderWidth: 1,
    borderColor: "#7f1d1d",
    padding: 12,
    gap: 8,
    zIndex: 15,
  },
  errorText: {
    color: "#fecaca",
    fontSize: 12,
    lineHeight: 17,
    fontWeight: "500",
  },
  fallbackLink: {
    color: "#93c5fd",
    fontSize: 13,
    fontWeight: "700",
  },
  centered: {
    flex: 1,
    backgroundColor: "#0b0b0f",
    alignItems: "center",
    justifyContent: "center",
    padding: 20,
  },
  card: {
    width: "100%",
    maxWidth: 400,
    borderRadius: 20,
    borderWidth: 1,
    borderColor: "rgba(255,255,255,0.1)",
    backgroundColor: "rgba(15,23,42,0.9)",
    padding: 22,
    gap: 12,
  },
  cardTitle: {
    color: "#fff",
    fontSize: 20,
    fontWeight: "700",
    textAlign: "center",
  },
  cardBody: {
    color: "#cbd5e1",
    fontSize: 14,
    lineHeight: 20,
    textAlign: "center",
  },
  primaryBtn: {
    marginTop: 8,
    borderRadius: 999,
    backgroundColor: "#6366f1",
    paddingVertical: 12,
    alignItems: "center",
  },
  primaryBtnText: {
    color: "#fff",
    fontWeight: "700",
    fontSize: 14,
  },
  secondaryBtn: {
    borderRadius: 999,
    paddingVertical: 10,
    alignItems: "center",
  },
  secondaryBtnText: {
    color: "#94a3b8",
    fontWeight: "600",
    fontSize: 14,
  },
});
