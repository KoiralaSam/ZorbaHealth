import { Ionicons } from "@expo/vector-icons";
import { registerGlobals } from "@livekit/react-native";
import { CameraView, useCameraPermissions, type BarcodeScanningResult } from "expo-camera";
import * as Location from "expo-location";
import * as SecureStore from "expo-secure-store";
import { StatusBar } from "expo-status-bar";
import { Room, RoomEvent } from "livekit-client";
import QRCode from "qrcode";
import React, {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import {
  ActivityIndicator,
  Alert,
  Image,
  KeyboardAvoidingView,
  Linking,
  Platform,
  Pressable,
  RefreshControl,
  ScrollView,
  Text,
  TextInput,
  View,
} from "react-native";
import {
  SafeAreaProvider,
  SafeAreaView,
} from "react-native-safe-area-context";
import {
  EmptyText,
  Feedback,
  Field,
  IconButton,
  InfoCard,
  LoadingCard,
  PrimaryButton,
  ScreenHeading,
  Section,
  Segmented,
  TabBar,
  TextButton,
} from "./src/components/primitives";
import { resolveIANATimezone } from "./src/timezone";
import { styles } from "./src/theme/styles";

const API_URL = process.env.EXPO_PUBLIC_API_URL ?? "http://localhost:8081";
const LOCATION_WS_URL =
  process.env.EXPO_PUBLIC_LOCATION_WS_URL ?? defaultLocationWSURL(API_URL);
const LOCATION_WS_RECONNECT_MS = 3000;

// Expo Go doesn't ship the @livekit/react-native-webrtc native module; only a
// development build does. Degrade gracefully so the rest of the app still runs.
try {
  registerGlobals();
} catch {
  console.warn(
    "LiveKit WebRTC native module unavailable (Expo Go?). Voice/video calls are disabled; build a dev client (eas build --profile development --platform android) for calls.",
  );
}

function defaultLocationWSURL(apiURL: string) {
  try {
    const url = new URL(apiURL);
    url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
    url.port = "8091";
    url.pathname = "";
    url.search = "";
    url.hash = "";
    return url.toString().replace(/\/$/, "");
  } catch {
    return "ws://localhost:8091";
  }
}

function locationWSBaseURL() {
  return LOCATION_WS_URL.replace(/\/$/, "");
}

function locationWSDisplayName() {
  return locationWSBaseURL().replace(/^wss?:\/\//, "");
}

function locationWSURLWithToken(token: string) {
  return `${locationWSBaseURL()}/ws/location?token=${encodeURIComponent(token)}`;
}

const endpoints = {
  patientLogin: "/api/v1/auth/patient/login",
  patientRefresh: "/api/v1/auth/patient/refresh",
  patientLogout: "/api/v1/auth/patient/logout",
  patientRegister: "/api/v1/auth/patient/register",
  patientVerifyEmail: "/api/v1/auth/patient/register/verify",
  patientVerifyOtp: "/api/v1/auth/patient/register/verify-otp",
  patientProfile: "/api/v1/patient/profile",
  patientConsents: "/api/v1/patient/consents",
  patientHospitalConsents: "/api/v1/patient/hospital-consents",
  patientConsentRequests: "/api/v1/patient/consent-requests",
  patientRecordsAnswer: "/api/v1/patient/records/answer",
  patientCalls: "/api/v1/patient/calls",
  patientBridgedCallTransfer: "/api/v1/patient/calls/bridge-transfer",
  patientBridgedCallSession: "/api/v1/patient/calls/bridge-session",
  patientBridgedCallTranslation: "/api/v1/patient/calls/bridge-translation",
  patientBridgedCallEnd: "/api/v1/patient/calls/bridge-end",
  patientAudit: "/api/v1/patient/audit",
  patientMeetings: "/api/v1/patient/meetings",
  patientSchedulableStaff: "/api/v1/patient/schedulable-staff",
  patientWelfareChecks: "/api/v1/patient/welfare-checks",
  hospitalLogin: "/api/v1/auth/hospital/login",
  hospitalRefresh: "/api/v1/auth/hospital/refresh",
  hospitalLogout: "/api/v1/auth/hospital/logout",
  hospitalPatients: "/api/v1/hospital/patients",
  hospitalSummary: "/api/v1/hospital/records/summary",
  hospitalIncidents: "/api/v1/hospital/incidents",
  hospitalPatientAudit: "/api/v1/hospital/patient/audit",
  hospitalBridgedCallConnect: "/api/v1/hospital/calls/bridge-connect",
  hospitalBridgedCallSession: "/api/v1/hospital/calls/bridge-session",
  hospitalBridgedCallSessions: "/api/v1/hospital/calls/bridge-sessions",
  hospitalBridgedCallTranslation: "/api/v1/hospital/calls/bridge-translation",
  hospitalBridgedCallEnd: "/api/v1/hospital/calls/bridge-end",
  hospitalMeetings: "/api/v1/hospital/meetings",
  hospitalStaffRegister: "/api/v1/auth/hospital/staff/register",
  hospitalConsentRequests: "/api/v1/hospital/consent-requests",
};

type Role = "patient" | "hospital";
type PatientTab =
  | "home"
  | "consents"
  | "scan"
  | "records"
  | "calls"
  | "meetings"
  | "welfare"
  | "audit"
  | "location";
type HospitalTab =
  | "home"
  | "summary"
  | "meetings"
  | "staff"
  | "consent"
  | "incidents"
  | "audit";

type APIError = { code: string; message: string };
type APIResponse<T> = { data?: T; error?: APIError };
class RequestError extends Error {
  code?: string;

  constructor(message: string, code?: string) {
    super(message);
    this.name = "RequestError";
    this.code = code;
  }
}

const MOBILE_CACHE_TTL_MS = 45_000;
const mobileApiCache = new Map<
  string,
  {
    expiresAt: number;
    value?: unknown;
    promise?: Promise<unknown>;
  }
>();

function mobileCacheKey(endpoint: string, token?: string, role?: Role) {
  return `${role ?? "public"}:${(token ?? "").slice(-16)}:${endpoint}`;
}

function clearMobileApiCache(role?: Role) {
  if (!role) {
    mobileApiCache.clear();
    return;
  }
  for (const key of mobileApiCache.keys()) {
    if (key.startsWith(`${role}:`)) mobileApiCache.delete(key);
  }
}

type PatientLoginData = {
  message?: string;
  access_token?: string;
  patient_id?: string;
  refresh_token?: string;
};
type HospitalLoginData = {
  message?: string;
  access_token?: string;
  hospital_id?: string;
  staff_id?: string;
  role?: string;
  refresh_token?: string;
};
type RefreshData = {
  access_token?: string;
  refresh_token?: string;
};
type PatientProfile = {
  patient_id?: string;
  full_name?: string;
  email?: string;
  phone_number?: string;
  date_of_birth?: string;
  medical_notes?: string;
  voice_phone?: string;
  voice_enabled?: boolean;
  support_window?: string;
};
type ConsentRecord = {
  consent_id?: string;
  consent_type?: string;
  granted_at?: string;
  revoked_at?: string;
  scope?: string;
  source?: string;
  status?: string;
};
type Citation = { text?: string; source_file?: string; score?: number };
type CallSummary = {
  id: number;
  status?: string;
  started_at?: string;
  ended_at?: string;
  summary?: string;
  recording_url?: string;
  livekit_room_id?: string;
};

function resolveLiveCallSessionId(calls: CallSummary[]): string {
  const active = calls.find(
    (call) =>
      call.status?.toLowerCase() === "active" &&
      Boolean(call.livekit_room_id?.trim()),
  );
  if (active?.livekit_room_id?.trim()) {
    return active.livekit_room_id.trim();
  }
  const open = calls.find(
    (call) => Boolean(call.livekit_room_id?.trim()) && !call.ended_at,
  );
  return open?.livekit_room_id?.trim() || "";
}
type BridgedCallTranslationPreferences = {
  enabled?: boolean;
  language_mode?: string;
  language_code?: string;
  participant_identity?: string;
  updated_at?: string;
};
type BridgedCallSession = {
  session_id?: string;
  room_sid?: string;
  patient_id?: string;
  hospital_id?: string;
  staff_id?: string;
  status?: string;
  requested_at?: string;
  connected_at?: string;
  ended_at?: string;
  transfer_reason?: string;
  patient_translation?: BridgedCallTranslationPreferences;
  staff_translation?: BridgedCallTranslationPreferences;
};
type BridgedCallSessionResponseData = {
  session?: BridgedCallSession;
  patient_room_token?: string;
  staff_room_token?: string;
  livekit_ws_url?: string;
};
type InterpretationSegmentMessage = {
  type?: string;
  participant?: string;
  source_language?: string;
  target_language?: string;
  original_text?: string;
  translated_text?: string;
  passthrough?: boolean;
};
type AuditEvent = {
  event_id?: string;
  event_type?: string;
  actor_type?: string;
  actor_id?: string;
  patient_id?: string;
  service_name?: string;
  resource_type?: string;
  resource_id?: string;
  timestamp?: string;
  correlation_id?: string;
  tool_name?: string;
  success_status?: boolean;
  failure_reason?: string;
  metadata?: Record<string, unknown>;
};
type Incident = {
  event_id?: string;
  patient_id?: string;
  timestamp?: string;
  severity?: string;
  session_id?: string;
  service_name?: string;
  failure_reason?: string;
  metadata?: Record<string, unknown>;
};

function bridgeCaptionLabel(participant?: string) {
  return participant === "staff" ? "Clinician" : "Patient";
}

function parseMeetingJoinURL(joinURL: string): { server: string; room: string; token: string } | null {
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

function MeetingVideoRoom({
  joinURL,
  onClose,
}: {
  joinURL: string;
  onClose: () => void;
}) {
  const [status, setStatus] = useState("Connecting…");
  const [error, setError] = useState("");
  const roomRef = useRef<Room | null>(null);

  useEffect(() => {
    const parsed = parseMeetingJoinURL(joinURL);
    if (!parsed) {
      setError("This meeting link is missing LiveKit join details. Open it in the browser instead.");
      setStatus("");
      return;
    }
    const room = new Room({ adaptiveStream: true, dynacast: true });
    roomRef.current = room;
    room.on(RoomEvent.Disconnected, () => setStatus("Disconnected"));
    let cancelled = false;
    (async () => {
      try {
        await room.connect(parsed.server, parsed.token);
        if (cancelled) return;
        setStatus(parsed.room ? `Connected — ${parsed.room}` : "Connected");
        await room.localParticipant.setCameraEnabled(true).catch(() => undefined);
        await room.localParticipant.setMicrophoneEnabled(true).catch(() => undefined);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Unable to join the video visit.");
          setStatus("");
        }
      }
    })();
    return () => {
      cancelled = true;
      void room.disconnect();
      roomRef.current = null;
    };
  }, [joinURL]);

  return (
    <Section title="Video visit">
      {status ? <Text style={styles.meta}>{status}</Text> : null}
      <Feedback error={error} />
      <Text style={styles.cardBody}>
        Camera and microphone are enabled for this LiveKit visit. Stay on this screen until the visit ends.
      </Text>
      <View style={styles.inlineActions}>
        <IconButton
          icon="call-outline"
          label="Leave"
          onPress={() => {
            void roomRef.current?.disconnect();
            onClose();
          }}
          tone="neutral"
        />
        {error ? (
          <IconButton
            icon="open-outline"
            label="Open in browser"
            onPress={() => Linking.openURL(joinURL)}
            tone="accent"
          />
        ) : null}
      </View>
    </Section>
  );
}
type HospitalPatient = {
  patient_id?: string;
  full_name?: string;
  email?: string;
  phone_number?: string;
  date_of_birth?: string;
  consent_granted_at?: string;
  last_call_at?: string;
};
type HospitalMeeting = {
  id: string;
  patient_id?: string;
  staff_id?: string;
  hospital_id?: string;
  starts_at?: string;
  duration_minutes?: number;
  timezone?: string;
  title?: string;
  join_url?: string;
  status?: string;
  correlation_id?: string;
};
type PatientSchedulableStaffMember = {
  staff_id: string;
  hospital_id: string;
  name: string;
  role: string;
  email: string;
};

type WelfareCheckReason =
  | "medication_reminder"
  | "mental_wellbeing"
  | "daily_checkup"
  | "symptom_follow_up"
  | "care_plan_reminder"
  | "other";

type PatientWelfareCheck = {
  id: string;
  patient_id?: string;
  scheduled_at?: string;
  timezone?: string;
  reason_code?: WelfareCheckReason | string;
  reason_detail?: string;
  status?: string;
  created_at?: string;
};

const welfareReasonLabels: Record<WelfareCheckReason, string> = {
  medication_reminder: "Medication reminder",
  mental_wellbeing: "Mental wellbeing",
  daily_checkup: "Daily checkup",
  symptom_follow_up: "Symptom follow-up",
  care_plan_reminder: "Care plan reminder",
  other: "Other",
};

const welfareReasons = Object.keys(welfareReasonLabels) as WelfareCheckReason[];

type HospitalConsentRequest = {
  id?: string;
  token?: string;
  hospital_id?: string;
  hospital_name?: string;
  staff_id?: string;
  staff_name?: string;
  staff_role?: string;
  patient_id?: string;
  requested_permissions?: string[];
  note?: string;
  expires_at?: string;
  approved_at?: string;
  created_at?: string;
  status?: string;
  qr_payload?: string;
};
type PatientHospitalConsent = {
  hospital_id?: string;
  hospital_name?: string;
  granted_at?: string;
  revoked_at?: string;
  status?: string;
};

const auditEventTypes = [
  "PATIENT_CREATED",
  "PATIENT_VERIFIED",
  "PATIENT_LOGIN",
  "PATIENT_LOGOUT",
  "HEALTH_RECORD_CREATED",
  "HEALTH_RECORD_VIEWED",
  "HEALTH_RECORD_SEARCHED",
  "HEALTH_RECORD_SUMMARIZED",
  "AI_TOOL_CALLED",
  "AI_RESPONSE_GENERATED",
  "LOCATION_REQUESTED",
  "EMERGENCY_ESCALATION_TRIGGERED",
  "NOTIFICATION_SENT",
  "CONSENT_GRANTED",
  "CONSENT_REVOKED",
  "TRANSLATION_REQUESTED",
] as const;

function getAuditEventTypeOptions(events: AuditEvent[]) {
  return Array.from(
    new Set([
      ...auditEventTypes,
      ...events.map((event) => event.event_type).filter(Boolean),
    ]),
  ) as string[];
}

const consentCopy: Record<string, { label: string; description: string }> = {
  VOICE_ASSISTANT_USE: {
    label: "Voice assistant",
    description: "Allows Zorba to support phone-based care conversations.",
  },
  HEALTH_RECORD_ACCESS: {
    label: "Health records",
    description: "Allows answers and summaries to reference your records.",
  },
  LOCATION_ACCESS: {
    label: "Emergency location",
    description: "Shares GPS only during active emergency voice sessions.",
  },
  SMS_NOTIFICATION: {
    label: "SMS notifications",
    description: "Allows important care updates by text message.",
  },
  EMAIL_NOTIFICATION: {
    label: "Email notifications",
    description: "Allows care updates and verification messages by email.",
  },
  AI_SUMMARIZATION: {
    label: "AI summaries",
    description: "Allows Zorba to create concise clinical summaries.",
  },
  THIRD_PARTY_MODEL_PROCESSING: {
    label: "Model processing",
    description: "Allows approved model providers to process limited context.",
  },
};

const consentTypes = Object.keys(consentCopy);
const focusOptions = ["full", "medications", "allergies", "diagnoses"];

async function saveSecure(key: string, value: string) {
  await SecureStore.setItemAsync(key, value);
}

async function readSecure(key: string) {
  return SecureStore.getItemAsync(key);
}

async function deleteSecure(key: string) {
  await SecureStore.deleteItemAsync(key);
}

async function apiRequest<T>(
  endpoint: string,
  options: {
    method?: string;
    token?: string;
    body?: unknown;
    role?: Role;
    skipAuthRetry?: boolean;
    cacheTTL?: number;
    forceRefresh?: boolean;
  } = {},
): Promise<T> {
  const method = options.method ?? "GET";
  const canUseCache = method === "GET" && Boolean(options.cacheTTL);
  const key = mobileCacheKey(endpoint, options.token, options.role);
  const now = Date.now();
  const cached = mobileApiCache.get(key);
  if (canUseCache && !options.forceRefresh && cached?.value !== undefined && cached.expiresAt > now) {
    return cached.value as T;
  }
  if (canUseCache && !options.forceRefresh && cached?.promise) {
    return cached.promise as Promise<T>;
  }

  const requestPromise = performApiRequest<T>(endpoint, options, method);
  if (canUseCache) {
    mobileApiCache.set(key, {
      expiresAt: now + (options.cacheTTL ?? MOBILE_CACHE_TTL_MS),
      promise: requestPromise,
    });
  }

  try {
    const value = await requestPromise;
    if (canUseCache) {
      mobileApiCache.set(key, {
        expiresAt: Date.now() + (options.cacheTTL ?? MOBILE_CACHE_TTL_MS),
        value,
      });
    } else if (method !== "GET") {
      clearMobileApiCache(options.role);
    }
    return value;
  } catch (error) {
    if (canUseCache) mobileApiCache.delete(key);
    throw error;
  }
}

async function performApiRequest<T>(
  endpoint: string,
  options: {
    method?: string;
    token?: string;
    body?: unknown;
    role?: Role;
    skipAuthRetry?: boolean;
    cacheTTL?: number;
    forceRefresh?: boolean;
  },
  method: string,
): Promise<T> {
  const headers: Record<string, string> = {
    Accept: "application/json",
    "X-Zorba-Client": "mobile",
  };
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (options.token) {
    headers.Authorization = `Bearer ${options.token}`;
  }

  const response = await fetch(`${API_URL}${endpoint}`, {
    method,
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
  });
  const payload = (await response.json()) as APIResponse<T>;
  if (response.status === 401 && options.role && !options.skipAuthRetry) {
    const refreshed = await silentRefresh(options.role);
    if (refreshed) {
      return apiRequest<T>(endpoint, {
        ...options,
        token: refreshed,
        skipAuthRetry: true,
      });
    }
  }
  if (!response.ok) {
    throw new RequestError(
      payload.error?.message ?? "Request failed.",
      payload.error?.code,
    );
  }
  return payload.data ?? ({} as T);
}

async function silentRefresh(role: Role): Promise<string | null> {
  const refreshKey =
    role === "patient" ? "patient_refresh_token" : "hospital_refresh_token";
  const refresh = await readSecure(refreshKey);
  if (!refresh) return null;
  const path =
    role === "patient" ? endpoints.patientRefresh : endpoints.hospitalRefresh;
  try {
    const data = await apiRequest<RefreshData>(path, {
      method: "POST",
      token: refresh,
      skipAuthRetry: true,
    });
    if (data.refresh_token) await saveSecure(refreshKey, data.refresh_token);
    return data.access_token ?? null;
  } catch (err) {
    if (err instanceof RequestError && err.code === "REFRESH_TOKEN_REUSE") {
      await deleteSecure(refreshKey);
      if (role === "patient") await deleteSecure("patient_id");
    }
    return null;
  }
}

function formatTime(value?: string) {
  const date = meaningfulDate(value);
  if (!date) return "Unknown time";
  return date.toLocaleString();
}

function meaningfulDate(value?: string) {
  if (!value) return null;
  const date = new Date(value);
  if (Number.isNaN(date.getTime()) || date.getFullYear() < 2000) return null;
  return date;
}

function titleFromCode(value?: string) {
  return value ? value.replaceAll("_", " ").toLowerCase() : "Unknown";
}

function parseFilterDate(value: string) {
  if (!value.trim()) return null;
  const normalized = value.trim().replace(" ", "T");
  const date = new Date(normalized);
  if (Number.isNaN(date.getTime())) return null;
  return date;
}

function normalizeCalls(data: {
  calls?: CallSummary[];
  items?: CallSummary[];
}) {
  const calls = data.calls ?? data.items ?? [];
  return [...calls].sort((a, b) => {
    const aTime = meaningfulDate(a.started_at)?.getTime() ?? 0;
    const bTime = meaningfulDate(b.started_at)?.getTime() ?? 0;
    return bTime - aTime;
  });
}

function filterAuditEvents(
  events: AuditEvent[],
  type: string,
  from: string,
  to: string,
) {
  const fromTime = parseFilterDate(from)?.getTime();
  const toTime = parseFilterDate(to)?.getTime();

  return events.filter((event) => {
    if (type !== "all" && event.event_type !== type) return false;
    const eventDate = meaningfulDate(event.timestamp);
    if ((fromTime || toTime) && !eventDate) return false;
    const eventTime = eventDate?.getTime();
    if (fromTime && eventTime !== undefined && eventTime < fromTime)
      return false;
    if (toTime && eventTime !== undefined && eventTime > toTime) return false;
    return true;
  });
}

export default function App() {
  const [role, setRole] = useState<Role>("patient");
  const [patientToken, setPatientToken] = useState("");
  const [hospitalToken, setHospitalToken] = useState("");
  const [booting, setBooting] = useState(true);

  useEffect(() => {
    const load = async () => {
      const [patientAccess, hospitalAccess] = await Promise.all([
        silentRefresh("patient"),
        silentRefresh("hospital"),
      ]);
      setPatientToken(patientAccess ?? "");
      setHospitalToken(hospitalAccess ?? "");
      if (hospitalAccess && !patientAccess) {
        setRole("hospital");
      }
      setBooting(false);
    };
    void load();
  }, []);

  const signOut = async () => {
    try {
      if (patientToken) {
        await apiRequest(endpoints.patientLogout, {
          method: "POST",
          token: patientToken,
          skipAuthRetry: true,
        });
      }
      if (hospitalToken) {
        await apiRequest(endpoints.hospitalLogout, {
          method: "POST",
          token: hospitalToken,
          skipAuthRetry: true,
        });
      }
    } catch {
      // ignore logout errors
    }
    await Promise.all([
      deleteSecure("patient_refresh_token"),
      deleteSecure("patient_id"),
      deleteSecure("hospital_refresh_token"),
    ]);
    clearMobileApiCache();
    setPatientToken("");
    setHospitalToken("");
  };

  return (
    <SafeAreaProvider>
      <SafeAreaView style={styles.safeArea} edges={["top", "right", "bottom", "left"]}>
        <StatusBar style="dark" />
        {booting ? (
          <View style={styles.center}>
            <ActivityIndicator />
            <Text style={styles.muted}>Loading secure session...</Text>
          </View>
        ) : (
          <KeyboardAvoidingView
            style={styles.shell}
            behavior={Platform.OS === "ios" ? "padding" : undefined}
          >
            <Header
              role={role}
              onRoleChange={setRole}
              signedIn={Boolean(patientToken || hospitalToken)}
              onSignOut={signOut}
            />
            {role === "patient" ? (
              patientToken ? (
                <PatientPortal token={patientToken} onSignOut={signOut} />
              ) : (
                <PatientAuth onLogin={setPatientToken} />
              )
            ) : hospitalToken ? (
              <HospitalPortal token={hospitalToken} onSignOut={signOut} />
            ) : (
              <HospitalAuth onLogin={setHospitalToken} />
            )}
          </KeyboardAvoidingView>
        )}
      </SafeAreaView>
    </SafeAreaProvider>
  );
}

function Header({
  role,
  onRoleChange,
  signedIn,
  onSignOut,
}: {
  role: Role;
  onRoleChange: (role: Role) => void;
  signedIn: boolean;
  onSignOut: () => void;
}) {
  return (
    <View style={styles.header}>
      <View style={styles.brandContainer}>
        <View style={styles.brandIconContainer}>
          <Ionicons
            name={role === "patient" ? "pulse" : "medkit"}
            size={22}
            color="#ffffff"
          />
        </View>
        <View>
          <Text style={styles.brand}>Zorba Health</Text>
          <Text style={styles.headerSub}>
            {role === "patient"
              ? "Patient mobile care"
              : "Clinician staff console"}
          </Text>
        </View>
      </View>
      <View style={styles.headerActions}>
        {!signedIn ? (
          <Segmented
            value={role}
            options={[
              { value: "patient", label: "Patient" },
              { value: "hospital", label: "Staff" },
            ]}
            onChange={(value) => onRoleChange(value as Role)}
          />
        ) : (
          <IconButton
            icon="log-out-outline"
            label="Sign out"
            onPress={onSignOut}
            tone="neutral"
          />
        )}
      </View>
    </View>
  );
}

function PatientAuth({ onLogin }: { onLogin: (token: string) => void }) {
  const [mode, setMode] = useState<"login" | "register" | "otp" | "email">(
    "login",
  );
  const [loginIdentifier, setLoginIdentifier] = useState("");
  const [phone, setPhone] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fullName, setFullName] = useState("");
  const [dateOfBirth, setDateOfBirth] = useState("");
  const [otp, setOtp] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const switchMode = (nextMode: "login" | "register") => {
    setMode(nextMode);
    setError("");
    setNotice("");
    if (nextMode === "register") {
      setOtp("");
    }
  };

  const login = async () => {
    if (!loginIdentifier.trim() || !password) {
      setError("Enter your email or phone number and password.");
      return;
    }

    setLoading(true);
    setError("");
    setNotice("");
    try {
      const data = await apiRequest<PatientLoginData>(endpoints.patientLogin, {
        method: "POST",
        body: { identifier: loginIdentifier.trim(), password },
      });
      if (!data.access_token)
        throw new Error("Login succeeded but no patient token was returned.");
      if (data.refresh_token)
        await saveSecure("patient_refresh_token", data.refresh_token);
      if (data.patient_id) await saveSecure("patient_id", data.patient_id);
      onLogin(data.access_token);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed.");
    } finally {
      setLoading(false);
    }
  };

  const register = async () => {
    const trimmedPhone = phone.trim();
    const trimmedEmail = email.trim();
    const trimmedFullName = fullName.trim();

    if (!trimmedPhone || !password || !trimmedFullName) {
      setError("Phone number, full name, and password are required.");
      return;
    }
    if (password.length < 8) {
      setError("Password must be at least 8 characters.");
      return;
    }

    setLoading(true);
    setError("");
    setNotice("");
    try {
      await apiRequest(endpoints.patientRegister, {
        method: "POST",
        body: {
          phone_number: trimmedPhone,
          email: trimmedEmail || undefined,
          password,
          full_name: trimmedFullName,
          date_of_birth: dateOfBirth
            ? new Date(dateOfBirth).toISOString()
            : undefined,
        },
      });
      setOtp("");
      setNotice("Registration started. Enter the OTP sent to your phone.");
      setMode("otp");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Registration failed.");
    } finally {
      setLoading(false);
    }
  };

  const verifyOtp = async () => {
    if (!phone.trim() || !otp.trim()) {
      setError("Enter the OTP sent to your phone.");
      return;
    }

    setLoading(true);
    setError("");
    setNotice("");
    try {
      await apiRequest(endpoints.patientVerifyOtp, {
        method: "POST",
        body: { phone_number: phone.trim(), otp: otp.trim() },
      });
      setNotice("Phone verified. Check your email for the verification link.");
      setMode("email");
    } catch (err) {
      setError(err instanceof Error ? err.message : "OTP verification failed.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <ScrollView
      contentContainerStyle={styles.authScroll}
      showsVerticalScrollIndicator={false}
    >
      <View style={styles.authPanel}>
        <Text style={styles.screenTitle}>
          {mode === "login"
            ? "Patient Sign In"
            : mode === "register"
              ? "Create Account"
              : mode === "otp"
                ? "Verify Phone"
                : "Check Your Email"}
        </Text>
        <Text style={styles.screenCopy}>
          {mode === "email"
            ? "Open the verification link sent to your email address, then return here to sign in."
            : "Access Zorba health record insights, voice assistant options, and clinic logs securely."}
        </Text>

        <View style={styles.stack}>
          {mode === "login" ? (
            <Field
              label="Email or phone number"
              value={loginIdentifier}
              onChangeText={setLoginIdentifier}
              autoCapitalize="none"
              placeholder="you@example.com or +15551234567"
            />
          ) : null}
          {mode !== "login" && mode !== "email" ? (
            <Field
              label="Phone number"
              value={phone}
              onChangeText={setPhone}
              keyboardType="phone-pad"
              placeholder="+15551234567"
            />
          ) : null}
          {mode === "register" ? (
            <Field
              label="Email Address"
              value={email}
              onChangeText={setEmail}
              keyboardType="email-address"
              autoCapitalize="none"
              placeholder="you@example.com"
            />
          ) : null}
          {mode === "login" || mode === "register" ? (
            <Field
              label="Password"
              value={password}
              onChangeText={setPassword}
              secureTextEntry
              placeholder="••••••••"
            />
          ) : null}
          {mode === "register" && (
            <>
              <Field
                label="Full name"
                value={fullName}
                onChangeText={setFullName}
                placeholder="John Doe"
              />
              <Field
                label="Date of birth"
                value={dateOfBirth}
                onChangeText={setDateOfBirth}
                placeholder="YYYY-MM-DD"
              />
            </>
          )}
          {mode === "otp" ? (
            <Field
              label="One-time code"
              value={otp}
              onChangeText={setOtp}
              keyboardType="number-pad"
              placeholder="6-digit code"
            />
          ) : null}
          {mode === "email" ? (
            <View style={styles.emailNoticeCard}>
              <Ionicons name="mail-unread-outline" size={24} color="#4f46e5" />
              <View style={styles.flex}>
                <Text style={styles.cardTitle}>
                  Email verification required
                </Text>
                <Text style={styles.cardBody}>
                  We sent a secure verification link to{" "}
                  {email.trim() || "your email address"}. After verification,
                  use your credentials to sign in.
                </Text>
              </View>
            </View>
          ) : null}
        </View>

        {mode === "email" ? (
          <PrimaryButton
            icon="log-in-outline"
            label="Go to Sign In"
            onPress={() => switchMode("login")}
          />
        ) : (
          <PrimaryButton
            icon={
              mode === "login" ? "log-in-outline" : "checkmark-circle-outline"
            }
            label={
              loading
                ? "Working..."
                : mode === "login"
                  ? "Sign In"
                  : mode === "register"
                    ? "Start Registration"
                    : "Verify OTP"
            }
            disabled={loading}
            onPress={
              mode === "login"
                ? login
                : mode === "register"
                  ? register
                  : verifyOtp
            }
          />
        )}

        <View style={styles.inlineActions}>
          <TextButton
            label={mode === "login" ? "Create account" : "Back to sign in"}
            onPress={() => switchMode(mode === "login" ? "register" : "login")}
          />
        </View>
        <Feedback error={error} notice={notice} />
      </View>
    </ScrollView>
  );
}

function HospitalAuth({ onLogin }: { onLogin: (token: string) => void }) {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const login = async () => {
    setLoading(true);
    setError("");
    try {
      const data = await apiRequest<HospitalLoginData>(
        endpoints.hospitalLogin,
        {
          method: "POST",
          body: { email: email.trim(), password },
        },
      );
      if (!data.access_token)
        throw new Error("Login succeeded but no hospital token was returned.");
      if (data.refresh_token)
        await saveSecure("hospital_refresh_token", data.refresh_token);
      onLogin(data.access_token);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Hospital login failed.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <ScrollView
      contentContainerStyle={styles.authScroll}
      showsVerticalScrollIndicator={false}
    >
      <View style={styles.authPanel}>
        <Text style={styles.screenTitle}>Hospital Sign In</Text>
        <Text style={styles.screenCopy}>
          Access clinical dashboards, monitor emergency incident indicators, and
          inspect patient logs.
        </Text>

        <View style={styles.stack}>
          <Field
            label="Clinical Email Address"
            value={email}
            onChangeText={setEmail}
            keyboardType="email-address"
            autoCapitalize="none"
            placeholder="staff@hospital.com"
          />
          <Field
            label="Password"
            value={password}
            onChangeText={setPassword}
            secureTextEntry
            placeholder="••••••••"
          />
        </View>

        <PrimaryButton
          icon="shield-checkmark-outline"
          label={loading ? "Signing in..." : "Sign In"}
          disabled={loading}
          onPress={login}
        />
        <Feedback error={error} />
      </View>
    </ScrollView>
  );
}

function PatientPortal({
  token,
  onSignOut,
}: {
  token: string;
  onSignOut: () => void;
}) {
  const [tab, setTab] = useState<PatientTab>("home");
  const [profile, setProfile] = useState<PatientProfile | null>(null);
  const [consents, setConsents] = useState<ConsentRecord[]>([]);
  const [hospitalConsents, setHospitalConsents] = useState<PatientHospitalConsent[]>([]);
  const [calls, setCalls] = useState<CallSummary[]>([]);
  const [audit, setAudit] = useState<AuditEvent[]>([]);
  const [meetings, setMeetings] = useState<HospitalMeeting[]>([]);
  const [welfareChecks, setWelfareChecks] = useState<PatientWelfareCheck[]>([]);
  const [schedulableStaff, setSchedulableStaff] = useState<PatientSchedulableStaffMember[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState("");
  const [showScheduleForm, setShowScheduleForm] = useState(false);
  const [scheduleStaffID, setScheduleStaffID] = useState("");
  const [scheduleHospitalID, setScheduleHospitalID] = useState("");
  const [scheduleStartsAt, setScheduleStartsAt] = useState("");
  const [scheduleDuration, setScheduleDuration] = useState("30");
  const [scheduleTimezone, setScheduleTimezone] = useState(() =>
    resolveIANATimezone(),
  );
  const [scheduleTitle, setScheduleTitle] = useState("");
  const [scheduleNotes, setScheduleNotes] = useState("");
  const [submittingSchedule, setSubmittingSchedule] = useState(false);
  const [mutatingMeetingID, setMutatingMeetingID] = useState("");
  const [welfareScheduledAt, setWelfareScheduledAt] = useState("");
  const [welfareReason, setWelfareReason] = useState<WelfareCheckReason>("daily_checkup");
  const [welfareDetail, setWelfareDetail] = useState("");
  const [creatingWelfareCheck, setCreatingWelfareCheck] = useState(false);
  const [cancellingWelfareCheck, setCancellingWelfareCheck] = useState<string | null>(null);

  const load = useCallback(async (showPageLoader = true) => {
    if (showPageLoader) {
      setLoading(true);
    } else {
      setRefreshing(true);
    }
    setError("");
    try {
      const cacheOptions = {
        cacheTTL: MOBILE_CACHE_TTL_MS,
        forceRefresh: !showPageLoader,
      };
      const [profileData, consentData, hospitalConsentData, callData, auditData, meetingsData, welfareData] = await Promise.all(
        [
          apiRequest<PatientProfile>(endpoints.patientProfile, {
            token,
            role: "patient",
            ...cacheOptions,
          }),
          apiRequest<{ consents?: ConsentRecord[] }>(
            endpoints.patientConsents,
            { token, role: "patient", ...cacheOptions },
          ),
          apiRequest<{ consents?: PatientHospitalConsent[] }>(
            endpoints.patientHospitalConsents,
            { token, role: "patient", ...cacheOptions },
          ),
          apiRequest<{ calls?: CallSummary[] }>(endpoints.patientCalls, {
            token,
            role: "patient",
            ...cacheOptions,
          }),
          apiRequest<{ events?: AuditEvent[] }>(endpoints.patientAudit, {
            token,
            role: "patient",
            ...cacheOptions,
          }),
          apiRequest<{ meetings?: HospitalMeeting[] }>(
            endpoints.patientMeetings,
            { token, role: "patient", ...cacheOptions },
          ),
          apiRequest<{ welfare_checks?: PatientWelfareCheck[] }>(
            endpoints.patientWelfareChecks,
            { token, role: "patient", ...cacheOptions },
          ),
        ],
      );
      setProfile(profileData);
      setConsents(consentData.consents ?? []);
      setHospitalConsents(hospitalConsentData.consents ?? []);
      setCalls(normalizeCalls(callData));
      setAudit(auditData.events ?? []);
      setMeetings(meetingsData.meetings ?? []);
      setWelfareChecks(welfareData.welfare_checks ?? []);
    } catch (err) {
      const message =
        err instanceof Error ? err.message : "Unable to load patient data.";
      setError(message);
      if (message.toLowerCase().includes("token")) onSignOut();
    } finally {
      if (showPageLoader) {
        setLoading(false);
      } else {
        setRefreshing(false);
      }
    }
  }, [onSignOut, token]);

  useEffect(() => {
    void load(true);
  }, [load]);

  useEffect(() => {
    void apiRequest<{ calls?: CallSummary[] }>(endpoints.patientCalls, {
      token,
      role: "patient",
      cacheTTL: 60_000,
    }).catch(() => undefined);
    void apiRequest<{ events?: AuditEvent[] }>(endpoints.patientAudit, {
      token,
      role: "patient",
      cacheTTL: 60_000,
    }).catch(() => undefined);
    void apiRequest<{ meetings?: HospitalMeeting[] }>(endpoints.patientMeetings, {
      token,
      role: "patient",
      cacheTTL: 60_000,
    }).catch(() => undefined);
  }, [token]);

  const refresh = useCallback(() => {
    void load(false);
  }, [load]);

  const loadSchedulableStaff = useCallback(async (hospitalID: string) => {
    if (!hospitalID) return;
    try {
      const data = await apiRequest<{ staff?: PatientSchedulableStaffMember[] }>(
        `${endpoints.patientSchedulableStaff}?hospital_id=${encodeURIComponent(hospitalID)}`,
        { token, role: "patient", cacheTTL: 5 * 60_000 },
      );
      setSchedulableStaff(data.staff ?? []);
    } catch {
      // best-effort
    }
  }, [token]);

  const handleScheduleMeeting = async () => {
    if (!token) return;
    const timezone = resolveIANATimezone(scheduleTimezone);
    if (timezone !== scheduleTimezone) {
      setScheduleTimezone(timezone);
    }
    setSubmittingSchedule(true);
    setError("");
    try {
      const data = await apiRequest<{ meeting?: HospitalMeeting }>(
        endpoints.patientMeetings,
        {
          method: "POST",
          token,
          role: "patient",
          body: {
            staff_id: scheduleStaffID,
            hospital_id: scheduleHospitalID,
            starts_at: new Date(scheduleStartsAt).toISOString(),
            duration_minutes: Number(scheduleDuration) || 30,
            timezone,
            title: scheduleTitle || undefined,
            notes: scheduleNotes || undefined,
          },
        },
      );
      if (data.meeting) setMeetings((prev) => [data.meeting!, ...prev]);
      setShowScheduleForm(false);
      setScheduleStaffID("");
      setScheduleStartsAt("");
      setScheduleTitle("");
      setScheduleNotes("");
      setError("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to schedule meeting.");
    } finally {
      setSubmittingSchedule(false);
    }
  };

  const handleCancelMeeting = async (meetingID: string) => {
    if (!token) return;
    setMutatingMeetingID(meetingID);
    setError("");
    try {
      const data = await apiRequest<{ meeting?: HospitalMeeting }>(
        `${endpoints.patientMeetings}/${encodeURIComponent(meetingID)}`,
        {
          method: "DELETE",
          token,
          role: "patient",
          body: { reason: "Cancelled by patient" },
        },
      );
      if (data.meeting) {
        setMeetings((prev) =>
          prev.map((m) => (m.id === meetingID ? data.meeting! : m)),
        );
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to cancel meeting.");
    } finally {
      setMutatingMeetingID("");
    }
  };

  const handleCreateWelfareCheck = async () => {
    if (!token || creatingWelfareCheck) return;
    if (!welfareScheduledAt.trim()) {
      setError("Choose a date and time for the welfare check.");
      return;
    }
    const scheduledAt = new Date(welfareScheduledAt);
    if (Number.isNaN(scheduledAt.getTime())) {
      setError("Welfare check time must be a valid date.");
      return;
    }
    const timezone = resolveIANATimezone(scheduleTimezone);
    if (timezone !== scheduleTimezone) {
      setScheduleTimezone(timezone);
    }
    setCreatingWelfareCheck(true);
    setError("");
    try {
      const data = await apiRequest<{ welfare_check?: PatientWelfareCheck }>(
        endpoints.patientWelfareChecks,
        {
          method: "POST",
          token,
          role: "patient",
          body: {
            scheduled_at: scheduledAt.toISOString(),
            timezone,
            reason_code: welfareReason,
            reason_detail: welfareDetail.trim() || undefined,
          },
        },
      );
      if (data.welfare_check) {
        setWelfareChecks((prev) => [data.welfare_check!, ...prev]);
      }
      setWelfareDetail("");
      setWelfareScheduledAt("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to schedule welfare check.");
    } finally {
      setCreatingWelfareCheck(false);
    }
  };

  const handleCancelWelfareCheck = async (id: string) => {
    if (!token || !id) return;
    setCancellingWelfareCheck(id);
    setError("");
    try {
      const data = await apiRequest<{ welfare_check?: PatientWelfareCheck }>(
        `${endpoints.patientWelfareChecks}/${encodeURIComponent(id)}`,
        { method: "DELETE", token, role: "patient" },
      );
      if (data.welfare_check) {
        setWelfareChecks((prev) =>
          prev.map((item) => (item.id === id ? data.welfare_check! : item)),
        );
      } else {
        setWelfareChecks((prev) =>
          prev.map((item) =>
            item.id === id ? { ...item, status: "cancelled" } : item,
          ),
        );
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to cancel welfare check.");
    } finally {
      setCancellingWelfareCheck(null);
    }
  };

  const activeConsents = useMemo(() => {
    const map = new Map<string, ConsentRecord>();
    for (const consent of consents) {
      if (
        consent.consent_type &&
        !consent.revoked_at &&
        consent.status !== "revoked"
      ) {
        map.set(consent.consent_type, consent);
      }
    }
    return map;
  }, [consents]);

  return (
    <View style={styles.portal}>
      <TabBar
        value={tab}
        onChange={(value) => setTab(value as PatientTab)}
        options={[
          { value: "home", label: "Home", icon: "home-outline" },
          { value: "consents", label: "Consent", icon: "options-outline" },
          { value: "scan", label: "Scan", icon: "qr-code-outline" },
          { value: "records", label: "Ask", icon: "chatbubbles-outline" },
          { value: "calls", label: "Calls", icon: "call-outline" },
          { value: "meetings", label: "Meetings", icon: "calendar-outline" },
          { value: "welfare", label: "Welfare", icon: "heart-outline" },
          { value: "audit", label: "Audit", icon: "reader-outline" },
          { value: "location", label: "GPS", icon: "navigate-outline" },
        ]}
      />
      <ScrollView
        contentContainerStyle={styles.content}
        showsVerticalScrollIndicator={false}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={refresh}
            tintColor="#4f46e5"
            colors={["#4f46e5"]}
          />
        }
      >
        {loading ? <LoadingCard /> : null}
        <Feedback error={error} />
        {!loading && tab === "home" ? (
          <PatientHome
            profile={profile}
            calls={calls}
            audit={audit}
            onRefresh={refresh}
          />
        ) : null}
        {!loading && tab === "consents" ? (
          <ConsentCenter
            token={token}
            consents={consents}
            setConsents={setConsents}
            hospitalConsents={hospitalConsents}
            setHospitalConsents={setHospitalConsents}
          />
        ) : null}
        {!loading && tab === "scan" ? (
          <PatientConsentScanner
            token={token}
            setHospitalConsents={setHospitalConsents}
          />
        ) : null}
        {!loading && tab === "records" ? (
          <HealthQuestion token={token} />
        ) : null}
        {!loading && tab === "calls" ? (
          <CallList
            token={token}
            calls={calls}
            voicePhone={profile?.voice_phone}
            hospitalConsents={hospitalConsents}
          />
        ) : null}
        {!loading && tab === "meetings" ? (
          <PatientMeetings
            token={token}
            meetings={meetings}
            setMeetings={setMeetings}
            schedulableStaff={schedulableStaff}
            hospitalConsents={hospitalConsents}
            showScheduleForm={showScheduleForm}
            setShowScheduleForm={setShowScheduleForm}
            scheduleStaffID={scheduleStaffID}
            setScheduleStaffID={setScheduleStaffID}
            scheduleHospitalID={scheduleHospitalID}
            setScheduleHospitalID={setScheduleHospitalID}
            scheduleStartsAt={scheduleStartsAt}
            setScheduleStartsAt={setScheduleStartsAt}
            scheduleDuration={scheduleDuration}
            setScheduleDuration={setScheduleDuration}
            scheduleTimezone={scheduleTimezone}
            setScheduleTimezone={setScheduleTimezone}
            scheduleTitle={scheduleTitle}
            setScheduleTitle={setScheduleTitle}
            scheduleNotes={scheduleNotes}
            setScheduleNotes={setScheduleNotes}
            submittingSchedule={submittingSchedule}
            mutatingMeetingID={mutatingMeetingID}
            onSchedule={handleScheduleMeeting}
            onCancel={handleCancelMeeting}
            onLoadSchedulableStaff={loadSchedulableStaff}
            onRefresh={() => void load(false)}
          />
        ) : null}
        {!loading && tab === "welfare" ? (
          <PatientWelfareChecks
            checks={welfareChecks}
            scheduledAt={welfareScheduledAt}
            setScheduledAt={setWelfareScheduledAt}
            reason={welfareReason}
            setReason={setWelfareReason}
            detail={welfareDetail}
            setDetail={setWelfareDetail}
            timezone={scheduleTimezone}
            creating={creatingWelfareCheck}
            cancellingId={cancellingWelfareCheck}
            onCreate={() => void handleCreateWelfareCheck()}
            onCancel={(id) => void handleCancelWelfareCheck(id)}
            onRefresh={() => void load(false)}
          />
        ) : null}
        {!loading && tab === "audit" ? <AuditList events={audit} /> : null}
        {!loading && tab === "location" ? (
          <LocationSharing
            token={token}
            enabled={activeConsents.has("LOCATION_ACCESS")}
          />
        ) : null}
      </ScrollView>
    </View>
  );
}

function PatientHome({
  profile,
  calls,
  audit,
  onRefresh,
}: {
  profile: PatientProfile | null;
  calls: CallSummary[];
  audit: AuditEvent[];
  onRefresh: () => void;
}) {
  return (
    <View style={styles.stack}>
      <View style={styles.heroPanel}>
        <View style={styles.rowBetween}>
          <Text style={styles.eyebrow}>Patient Home</Text>
          <Pressable onPress={onRefresh} style={styles.refreshIcon}>
            <Ionicons name="refresh" size={18} color="#e0e7ff" />
          </Pressable>
        </View>
        <Text style={styles.largeTitle}>
          {profile?.full_name || "Welcome Back"}
        </Text>
        <Text style={styles.heroCopy}>
          Zorba Voice Care: {profile?.voice_phone || "Support line active"}
        </Text>

        <View style={styles.heroActions}>
          <Pressable
            onPress={() => callNumber(profile?.voice_phone)}
            style={styles.heroCallBtn}
          >
            <Ionicons name="call" size={16} color="#4f46e5" />
            <Text style={styles.heroCallBtnText}>Call Assistant</Text>
          </Pressable>
        </View>
      </View>

      <View style={styles.gridTwo}>
        <InfoCard
          icon="person-outline"
          title="Demographics"
          body={`${profile?.phone_number || "No phone listed"}\n${profile?.email || "No email listed"}`}
        />
        <InfoCard
          icon="document-text-outline"
          title="Last Audit Event"
          body={
            audit[0]
              ? `${titleFromCode(audit[0].event_type)}\n${formatTime(audit[0].timestamp)}`
              : "No audit trail logs yet."
          }
        />
      </View>

      <Section title="Recent Calls">
        {calls.slice(0, 2).map((call) => (
          <CallRow key={String(call.id)} call={call} />
        ))}
        {calls.length === 0 ? (
          <EmptyText text="No call logs captured yet." />
        ) : null}
      </Section>
    </View>
  );
}

function ConsentCenter({
  token,
  consents,
  setConsents,
  hospitalConsents,
  setHospitalConsents,
}: {
  token: string;
  consents: ConsentRecord[];
  setConsents: (items: ConsentRecord[]) => void;
  hospitalConsents: PatientHospitalConsent[];
  setHospitalConsents: React.Dispatch<React.SetStateAction<PatientHospitalConsent[]>>;
}) {
  const [mutating, setMutating] = useState("");
  const [error, setError] = useState("");

  const active = useMemo(() => {
    const map = new Map<string, ConsentRecord>();
    for (const consent of consents) {
      if (
        consent.consent_type &&
        !consent.revoked_at &&
        consent.status !== "revoked"
      ) {
        map.set(consent.consent_type, consent);
      }
    }
    return map;
  }, [consents]);

  const mutate = async (type: string, grant: boolean) => {
    setMutating(type);
    setError("");
    try {
      const data = await apiRequest<{ consent?: ConsentRecord }>(
        endpoints.patientConsents,
        {
          method: grant ? "POST" : "DELETE",
          token,
          body: { consent_type: type, source: "patient-mobile-app" },
        },
      );
      if (data.consent) {
        setConsents([
          data.consent,
          ...consents.filter((item) => item.consent_type !== type),
        ]);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Consent update failed.");
    } finally {
      setMutating("");
    }
  };

  const revokeHospitalConsent = async (hospitalID?: string) => {
    if (!hospitalID) return;
    setMutating(hospitalID);
    setError("");
    try {
      await apiRequest(endpoints.patientHospitalConsents + `/${encodeURIComponent(hospitalID)}`, {
        method: "DELETE",
        token,
        role: "patient",
      });
      setHospitalConsents((current) =>
        current.map((item) =>
          item.hospital_id === hospitalID
            ? { ...item, status: "revoked", revoked_at: new Date().toISOString() }
            : item,
        ),
      );
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Hospital consent update failed.",
      );
    } finally {
      setMutating("");
    }
  };

  return (
    <View style={styles.stack}>
      <ScreenHeading
        title="Consent Center"
        subtitle="Manage permissions for voice processing, safety GPS, and record summarizers."
      />
      <Feedback error={error} />
      {consentTypes.map((type) => {
        const granted = active.has(type);
        return (
          <View
            key={type}
            style={[styles.card, granted ? styles.cardActive : null]}
          >
            <View style={styles.rowBetween}>
              <View style={styles.flex}>
                <Text style={styles.cardTitle}>{consentCopy[type].label}</Text>
                <Text style={styles.cardBody}>
                  {consentCopy[type].description}
                </Text>
              </View>
              <Pressable
                accessibilityRole="switch"
                accessibilityState={{ checked: granted }}
                disabled={mutating === type}
                onPress={() => mutate(type, !granted)}
                style={[
                  styles.switchTrack,
                  granted ? styles.switchOn : styles.switchOff,
                ]}
              >
                <View
                  style={[
                    styles.switchThumb,
                    granted ? styles.switchThumbOn : styles.switchThumbOff,
                  ]}
                />
              </Pressable>
            </View>
            <Text style={styles.meta}>
              {granted
                ? `Granted • ${formatTime(active.get(type)?.granted_at)}`
                : "Inactive"}
            </Text>
          </View>
        );
      })}
      <Section title="Hospital Access">
        {hospitalConsents.map((consent) => (
          <View key={consent.hospital_id} style={styles.card}>
            <View style={styles.rowBetween}>
              <View style={styles.flex}>
                <Text style={styles.cardTitle}>
                  {consent.hospital_name || consent.hospital_id}
                </Text>
                <Text style={styles.meta}>
                  {consent.status || "active"} • {formatTime(consent.granted_at)}
                </Text>
              </View>
              {consent.status !== "revoked" ? (
                <IconButton
                  icon="trash-outline"
                  label={mutating === consent.hospital_id ? "Revoking" : "Revoke"}
                  onPress={() => revokeHospitalConsent(consent.hospital_id)}
                  tone="neutral"
                />
              ) : null}
            </View>
          </View>
        ))}
        {hospitalConsents.length === 0 ? (
          <EmptyText text="No hospital has access to your patient record." />
        ) : null}
      </Section>
    </View>
  );
}

function PatientConsentScanner({
  token,
  setHospitalConsents,
}: {
  token: string;
  setHospitalConsents: React.Dispatch<React.SetStateAction<PatientHospitalConsent[]>>;
}) {
  const [permission, requestPermission] = useCameraPermissions();
  const [manualToken, setManualToken] = useState("");
  const [request, setRequest] = useState<HospitalConsentRequest | null>(null);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [scanned, setScanned] = useState(false);
  const [approving, setApproving] = useState(false);

  const lookup = async (rawValue: string) => {
    const requestToken = extractConsentToken(rawValue);
    if (!requestToken) return;
    setError("");
    setNotice("");
    try {
      const data = await apiRequest<{ request?: HospitalConsentRequest }>(
        `${endpoints.patientConsentRequests}/${encodeURIComponent(requestToken)}`,
        { token, role: "patient" },
      );
      setRequest(data.request ?? null);
      setManualToken(requestToken);
      setScanned(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Consent request lookup failed.");
      setScanned(false);
    }
  };

  const approve = async () => {
    const requestToken = request?.token || manualToken;
    if (!requestToken) return;
    setApproving(true);
    setError("");
    try {
      const data = await apiRequest<{
        message?: string;
        consent?: PatientHospitalConsent;
      }>(
        `${endpoints.patientConsentRequests}/${encodeURIComponent(requestToken)}/approve`,
        { method: "POST", token, role: "patient" },
      );
      if (data.consent) {
        setHospitalConsents((current) => [
          data.consent as PatientHospitalConsent,
          ...current.filter(
            (item) => item.hospital_id !== data.consent?.hospital_id,
          ),
        ]);
      }
      setNotice(data.message || "Hospital consent granted.");
      setRequest(null);
      setManualToken("");
      setScanned(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Consent approval failed.");
    } finally {
      setApproving(false);
    }
  };

  const onBarcodeScanned = ({ data }: BarcodeScanningResult) => {
    if (scanned) return;
    setScanned(true);
    void lookup(data);
  };

  return (
    <View style={styles.stack}>
      <ScreenHeading
        title="Scan Hospital QR"
        subtitle="Review hospital access requests from staff, then confirm consent from this portal."
      />
      <Feedback error={error} notice={notice} />
      {!permission?.granted ? (
        <PrimaryButton
          icon="camera-outline"
          label="Allow Camera"
          onPress={() => void requestPermission()}
        />
      ) : (
        <View style={styles.cameraFrame}>
          <CameraView
            style={styles.cameraPreview}
            facing="back"
            barcodeScannerSettings={{ barcodeTypes: ["qr"] }}
            onBarcodeScanned={scanned ? undefined : onBarcodeScanned}
          />
        </View>
      )}
      {scanned ? (
        <IconButton
          icon="refresh-outline"
          label="Scan another"
          onPress={() => {
            setScanned(false);
            setRequest(null);
          }}
          tone="neutral"
        />
      ) : null}
      <Field
        label="Manual QR Token"
        value={manualToken}
        onChangeText={setManualToken}
        autoCapitalize="none"
        placeholder="Paste token if camera is unavailable"
      />
      <PrimaryButton
        icon="search-outline"
        label="Lookup Consent Request"
        onPress={() => void lookup(manualToken)}
      />
      {request ? (
        <View style={[styles.card, styles.cardActive]}>
          <Text style={styles.eyebrow}>Pending consent request</Text>
          <Text style={styles.cardTitle}>
            {request.hospital_name || "Hospital access request"}
          </Text>
          <Text style={styles.cardBody}>
            Requested by {request.staff_name || "hospital staff"}
          </Text>
          {request.note ? (
            <Text style={styles.cardBody}>{request.note}</Text>
          ) : null}
          <Text style={styles.meta}>
            Expires {formatTime(request.expires_at)}
          </Text>
          <View style={styles.chipRow}>
            {(request.requested_permissions ?? []).map((permissionName) => (
              <Text key={permissionName} style={styles.badge}>
                {titleFromCode(permissionName)}
              </Text>
            ))}
          </View>
          <PrimaryButton
            icon="checkmark-circle-outline"
            label={approving ? "Approving..." : "Approve Hospital Access"}
            disabled={approving}
            onPress={approve}
          />
        </View>
      ) : null}
    </View>
  );
}

function HealthQuestion({ token }: { token: string }) {
  const [question, setQuestion] = useState("");
  const [answer, setAnswer] = useState("");
  const [citations, setCitations] = useState<Citation[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const ask = async () => {
    if (!question.trim()) return;
    setLoading(true);
    setError("");
    setNotice("");
    setAnswer("");
    setCitations([]);
    try {
      const data = await apiRequest<{
        answer?: string;
        citations?: Citation[];
      }>(endpoints.patientRecordsAnswer, {
        method: "POST",
        token,
        body: { question: question.trim(), top_k: 5 },
      });
      setAnswer(data.answer ?? "No answer returned.");
      setCitations(data.citations ?? []);
    } catch (err) {
      if (err instanceof RequestError && err.code === "NO_HEALTH_RECORDS") {
        setNotice(err.message);
        return;
      }
      setError(
        err instanceof Error ? err.message : "Unable to answer that question.",
      );
    } finally {
      setLoading(false);
    }
  };

  return (
    <View style={styles.stack}>
      <ScreenHeading
        title="Ask Your Records"
        subtitle="Query clinical files grounded in compliant record datasets."
      />
      <Field
        label="Inquiry Text"
        value={question}
        onChangeText={setQuestion}
        multiline
        placeholder="What medications are active in my record?"
      />
      <PrimaryButton
        icon="send-outline"
        label={loading ? "Analyzing..." : "Ask Zorba"}
        disabled={loading}
        onPress={ask}
      />
      <Feedback error={error} notice={notice} />

      {answer ? (
        <Section title="Clinician Answer">
          <View style={styles.card}>
            <Text style={styles.bodyText}>{answer}</Text>
          </View>
        </Section>
      ) : null}

      {citations.length > 0 ? (
        <Section title="Sources Referenced">
          {citations.map((citation, index) => (
            <View
              key={`${citation.source_file}-${index}`}
              style={styles.subCard}
            >
              <View
                style={{
                  flexDirection: "row",
                  alignItems: "center",
                  gap: 6,
                  marginBottom: 4,
                }}
              >
                <Ionicons
                  name="document-text-outline"
                  size={14}
                  color="#4f46e5"
                />
                <Text style={styles.cardTitle}>
                  {citation.source_file || "Record Source"}
                </Text>
              </View>
              <Text style={styles.cardBody}>
                {citation.text || "Snippet detail."}
              </Text>
            </View>
          ))}
        </Section>
      ) : null}
    </View>
  );
}

function CallList({
  token,
  calls,
  voicePhone,
  hospitalConsents,
}: {
  token: string;
  calls: CallSummary[];
  voicePhone?: string;
  hospitalConsents: PatientHospitalConsent[];
}) {
  const liveSessionId = useMemo(() => resolveLiveCallSessionId(calls), [calls]);
  const [hospitalID, setHospitalID] = useState(hospitalConsents[0]?.hospital_id || "");
  const [staffID, setStaffID] = useState("");
  const [enabled, setEnabled] = useState(true);
  const [languageMode, setLanguageMode] = useState("auto");
  const [languageCode, setLanguageCode] = useState("es");
  const [notice, setNotice] = useState("");
  const [session, setSession] = useState<BridgedCallSession | null>(null);
  const [busy, setBusy] = useState(false);
  const [bridgeRoomState, setBridgeRoomState] = useState<"disconnected" | "connecting" | "connected">("disconnected");
  const [bridgeCaptions, setBridgeCaptions] = useState<InterpretationSegmentMessage[]>([]);
  const bridgeRoomRef = useRef<Room | null>(null);

  useEffect(() => {
    if (hospitalID.trim() || hospitalConsents.length === 0) {
      return;
    }
    const first = hospitalConsents.find((item) => item.hospital_id)?.hospital_id;
    if (first) {
      setHospitalID(first);
    }
  }, [hospitalConsents, hospitalID]);

  const disconnectBridgeRoom = useCallback(async () => {
    const room = bridgeRoomRef.current;
    if (!room) {
      setBridgeRoomState("disconnected");
      return;
    }
    bridgeRoomRef.current = null;
    room.disconnect();
    setBridgeRoomState("disconnected");
  }, []);

  const joinBridgeRoom = useCallback(
    async (wsUrl?: string, tokenValue?: string) => {
      if (!wsUrl || !tokenValue) return;
      await disconnectBridgeRoom();
      const room = new Room();
      bridgeRoomRef.current = room;
      setBridgeRoomState("connecting");
      room.on(RoomEvent.DataReceived, (payload: Uint8Array, _participant, _kind, topic?: string) => {
        if (topic !== "zorba.interpretation") return;
        try {
          const message: InterpretationSegmentMessage = JSON.parse(
            new TextDecoder().decode(payload),
          );
          if (message.type !== "interpretation.segment") return;
          setBridgeCaptions((current) => [...current.slice(-49), message]);
        } catch {
          // Ignore malformed bridge packets.
        }
      });
      room.on(RoomEvent.Disconnected, () => {
        if (bridgeRoomRef.current === room) {
          bridgeRoomRef.current = null;
          setBridgeRoomState("disconnected");
        }
      });
      try {
        await room.connect(wsUrl, tokenValue);
        setBridgeRoomState("connected");
      } catch {
        setNotice("Interpreter started, but companion captions could not join.");
        await disconnectBridgeRoom();
      }
    },
    [disconnectBridgeRoom],
  );

  useEffect(() => () => {
    void disconnectBridgeRoom();
  }, [disconnectBridgeRoom]);

  const requestTransfer = async () => {
    if (!liveSessionId.trim()) {
      setNotice(
        "Start a verified call with Zorba first. We detect your active call automatically.",
      );
      return;
    }
    if (!hospitalID.trim()) {
      setNotice("Choose an approved hospital first.");
      return;
    }
    setBusy(true);
    setNotice("");
    try {
      const data = await apiRequest<BridgedCallSessionResponseData>(
        endpoints.patientBridgedCallTransfer,
        {
          method: "POST",
          token,
          role: "patient",
          body: {
            session_id: liveSessionId.trim(),
            room_sid: liveSessionId.trim(),
            hospital_id: hospitalID.trim(),
            staff_id: staffID.trim(),
            transfer_reason: "Patient requested live interpretation",
          },
        },
      );
      setSession(data.session ?? null);
      setBridgeCaptions([]);
      if (data.patient_room_token && data.livekit_ws_url) {
        await joinBridgeRoom(data.livekit_ws_url, data.patient_room_token);
      }
      setNotice("Hospital notified. A clinician can connect from their dashboard.");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Unable to request transfer.");
    } finally {
      setBusy(false);
    }
  };

  const refreshSession = async () => {
    if (!liveSessionId.trim()) {
      setNotice("No active phone call detected yet.");
      return;
    }
    setBusy(true);
    setNotice("");
    try {
      const data = await apiRequest<BridgedCallSessionResponseData>(
        `${endpoints.patientBridgedCallSession}?session_id=${encodeURIComponent(liveSessionId.trim())}`,
        { token, role: "patient" },
      );
      setSession(data.session ?? null);
      if (data.patient_room_token && data.livekit_ws_url) {
        await joinBridgeRoom(data.livekit_ws_url, data.patient_room_token);
      }
      setNotice(data.session ? "Interpreter status refreshed." : "No interpreter session yet.");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Unable to refresh session.");
    } finally {
      setBusy(false);
    }
  };

  const updateTranslation = async () => {
    if (!liveSessionId.trim()) {
      setNotice("No active phone call detected yet.");
      return;
    }
    setBusy(true);
    setNotice("");
    try {
      const data = await apiRequest<{ session?: BridgedCallSession }>(
        endpoints.patientBridgedCallTranslation,
        {
          method: "PUT",
          token,
          role: "patient",
          body: {
            session_id: liveSessionId.trim(),
            participant: "patient",
            translation: {
              enabled,
              language_mode: languageMode,
              language_code: languageCode.trim().toLowerCase(),
            },
          },
        },
      );
      setSession(data.session ?? null);
      setNotice("Patient translation preferences updated.");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Unable to update preferences.");
    } finally {
      setBusy(false);
    }
  };

  const endBridge = async () => {
    if (!liveSessionId.trim()) {
      setNotice("No active phone call detected yet.");
      return;
    }
    setBusy(true);
    setNotice("");
    try {
      const data = await apiRequest<{ session?: BridgedCallSession }>(
        endpoints.patientBridgedCallEnd,
        {
          method: "POST",
          token,
          role: "patient",
          body: {
            session_id: liveSessionId.trim(),
            reason: "Ended by patient",
          },
        },
      );
      setSession(data.session ?? null);
      setBridgeCaptions([]);
      await disconnectBridgeRoom();
      setNotice("Bridged call ended.");
    } catch (err) {
      setNotice(err instanceof Error ? err.message : "Unable to end bridged call.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <View style={styles.stack}>
      <ScreenHeading
        title="Call History Logs"
        subtitle="Consult automated session transcripts and summaries."
      />
      <PrimaryButton
        icon="call-outline"
        label="Call Assistance Hotline"
        onPress={() => callNumber(voicePhone)}
      />

      <Section title="Hospital interpreter">
        <Text style={styles.cardBody}>
          During a verified call with Zorba, request a hospital clinician with live translation. No session IDs to copy.
        </Text>
        <View
          style={[
            styles.card,
            liveSessionId
              ? { borderColor: "#6ee7b7", backgroundColor: "#ecfdf5" }
              : { borderColor: "#fcd34d", backgroundColor: "#fffbeb" },
          ]}
        >
          <Text style={styles.cardTitle}>
            {liveSessionId ? "Active call detected" : "No active call yet"}
          </Text>
          <Text style={styles.meta}>
            {liveSessionId
              ? "You can request an interpreter below."
              : "Call the hotline, complete verification, then return here."}
          </Text>
        </View>
        {hospitalConsents.length === 0 ? (
          <Text style={styles.meta}>Approve a hospital under Consents first.</Text>
        ) : (
          <>
            <Text style={styles.label}>Hospital</Text>
            <View style={styles.row}>
              {hospitalConsents.map((item) => {
                const id = item.hospital_id || "";
                if (!id) return null;
                const selected = hospitalID === id;
                return (
                  <Pressable
                    key={id}
                    onPress={() => setHospitalID(id)}
                    style={[styles.chip, selected ? styles.chipActive : null]}
                  >
                    <Text
                      style={[
                        styles.chipText,
                        selected ? styles.chipTextActive : null,
                      ]}
                    >
                      {item.hospital_name || "Hospital"}
                    </Text>
                  </Pressable>
                );
              })}
            </View>
          </>
        )}
        <Field
          label="Preferred clinician (optional)"
          value={staffID}
          onChangeText={setStaffID}
          placeholder="Any available staff"
        />
        <View style={styles.row}>
          <Pressable
            onPress={() => setEnabled(!enabled)}
            style={[styles.chip, enabled ? styles.chipActive : null]}
          >
            <Text style={[styles.chipText, enabled ? styles.chipTextActive : null]}>
              {enabled ? "Translation On" : "Translation Off"}
            </Text>
          </Pressable>
          <Pressable
            onPress={() =>
              setLanguageMode(languageMode === "auto" ? "manual" : "auto")
            }
            style={[
              styles.chip,
              languageMode === "auto" ? styles.chipActive : null,
            ]}
          >
            <Text
              style={[
                styles.chipText,
                languageMode === "auto" ? styles.chipTextActive : null,
              ]}
            >
              {languageMode === "auto" ? "Auto Detect" : "Manual"}
            </Text>
          </Pressable>
        </View>
        <Field
          label="Language you want to hear"
          value={languageCode}
          onChangeText={setLanguageCode}
          placeholder="es, ne, hi…"
        />
        <View style={styles.row}>
          <PrimaryButton
            icon="git-merge-outline"
            label={busy ? "Saving..." : "Request interpreter"}
            onPress={() => void requestTransfer()}
          />
          <IconButton
            icon="refresh-outline"
            label="Refresh"
            onPress={() => void refreshSession()}
            tone="neutral"
          />
          <IconButton
            icon="language-outline"
            label="Update Prefs"
            onPress={() => void updateTranslation()}
            tone="neutral"
          />
          <IconButton
            icon="exit-outline"
            label="End Bridge"
            onPress={() => void endBridge()}
            tone="neutral"
          />
        </View>
        {notice ? <Text style={styles.meta}>{notice}</Text> : null}
        {session ? (
          <View style={styles.card}>
            <Text style={styles.cardTitle}>Session Status: {session.status || "pending"}</Text>
            <Text style={styles.meta}>Hospital: {session.hospital_id || "unknown"}</Text>
            <Text style={styles.meta}>Staff: {session.staff_id || "awaiting assignment"}</Text>
            <Text style={styles.meta}>
              Your translation: {session.patient_translation?.enabled ? "enabled" : "disabled"} / {session.patient_translation?.language_mode || "auto"} / {session.patient_translation?.language_code || "default"}
            </Text>
            <Text style={styles.meta}>
              Companion captions: {bridgeRoomState}
            </Text>
            <Text style={styles.meta}>
              Interpreter mode: {session.status === "connected" ? "live" : "waiting for clinician"}
            </Text>
          </View>
        ) : null}
        {bridgeRoomState !== "disconnected" || bridgeCaptions.length > 0 ? (
          <View style={styles.card}>
            <Text style={styles.cardTitle}>Live Interpreter Captions</Text>
            <Text style={styles.meta}>
              Audio continues on the phone call. This screen mirrors interpreter captions only.
            </Text>
            {bridgeCaptions.length === 0 ? (
              <Text style={styles.meta}>Waiting for the patient or clinician to speak...</Text>
            ) : (
              bridgeCaptions.map((caption, index) => (
                <View key={`${caption.participant || "caption"}-${index}`} style={styles.subCard}>
                  <Text style={styles.badge}>{bridgeCaptionLabel(caption.participant)}</Text>
                  <Text style={styles.cardBody}>
                    {caption.translated_text || caption.original_text || "..."}
                  </Text>
                  {!caption.passthrough &&
                  caption.original_text &&
                  caption.original_text !== caption.translated_text ? (
                    <Text style={styles.meta}>{caption.original_text}</Text>
                  ) : null}
                </View>
              ))
            )}
          </View>
        ) : null}
      </Section>

      <Section title="Call Logs">
        {calls.map((call) => (
          <CallRow key={String(call.id)} call={call} />
        ))}
        {calls.length === 0 ? (
          <EmptyText text="No clinical calls archived." />
        ) : null}
      </Section>
    </View>
  );
}

function LocationSharing({
  token,
  enabled,
}: {
  token: string;
  enabled: boolean;
}) {
  const [status, setStatus] = useState(
    enabled
      ? "Waiting for voice session request."
      : "Grant emergency location consent first.",
  );
  const [connected, setConnected] = useState(false);
  const [activeSession, setActiveSession] = useState("");
  const wsRef = useRef<WebSocket | null>(null);
  const subscriptionRef = useRef<Location.LocationSubscription | null>(null);
  const reconnectRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const activeSessionRef = useRef("");

  const stopWatch = useCallback((clearSession = true) => {
    subscriptionRef.current?.remove();
    subscriptionRef.current = null;
    if (clearSession) {
      activeSessionRef.current = "";
      setActiveSession("");
    }
  }, []);

  const startLocationStream = useCallback(
    async (sessionID: string, ws: WebSocket) => {
      activeSessionRef.current = sessionID;
      setActiveSession(sessionID);

      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== "granted") {
        setStatus("GPS hardware access denied.");
        return;
      }

      stopWatch(false);

      const sendPosition = (position: Location.LocationObject) => {
        if (ws.readyState !== WebSocket.OPEN) return;
        ws.send(
          JSON.stringify({
            type: "location_update",
            sessionID,
            lat: position.coords.latitude,
            lng: position.coords.longitude,
            accuracy: position.coords.accuracy ?? 0,
            method: "gps",
          }),
        );
      };

      try {
        const current = await Location.getCurrentPositionAsync({
          accuracy: Location.Accuracy.High,
        });
        sendPosition(current);

        subscriptionRef.current = await Location.watchPositionAsync(
          {
            accuracy: Location.Accuracy.High,
            timeInterval: 5000,
            distanceInterval: 10,
          },
          sendPosition,
        );
        setStatus("Broadcasting safety GPS location stream.");
      } catch {
        setStatus("Could not read GPS location. Try again from the GPS tab.");
      }
    },
    [stopWatch],
  );

  useEffect(() => {
    let cancelled = false;

    const clearReconnect = () => {
      if (reconnectRef.current) {
        clearTimeout(reconnectRef.current);
        reconnectRef.current = null;
      }
    };

    if (!enabled) {
      clearReconnect();
      stopWatch();
      wsRef.current?.close();
      setConnected(false);
      setStatus("Grant emergency location consent first.");
      return;
    }

    const connect = () => {
      clearReconnect();
      const wsURL = locationWSURLWithToken(token);
      setStatus(`Connecting GPS channel to ${locationWSDisplayName()}...`);
      const ws = new WebSocket(wsURL);
      wsRef.current = ws;
      ws.onopen = () => {
        setConnected(true);
        const activeSessionID = activeSessionRef.current;
        if (activeSessionID) {
          void startLocationStream(activeSessionID, ws);
        } else {
          setStatus(
            "GPS is idle. Location coordinates route only when dispatch asks.",
          );
        }
      };
      ws.onclose = () => {
        setConnected(false);
        stopWatch(false);
        if (!cancelled) {
          setStatus(`GPS channel offline. Retrying ${locationWSDisplayName()}...`);
          reconnectRef.current = setTimeout(connect, LOCATION_WS_RECONNECT_MS);
        }
      };
      ws.onerror = () => {
        setStatus(
          `GPS channel error. Check EXPO_PUBLIC_LOCATION_WS_URL (${LOCATION_WS_URL}).`,
        );
      };
      ws.onmessage = async (event) => {
        const command = parseLocationCommand(event.data);
        if (!command) return;
        if (command.command === "stop_location") {
          stopWatch();
          setStatus("Location dispatch stopped.");
          return;
        }
        if (command.command === "start_location" && command.sessionID) {
          await startLocationStream(command.sessionID, ws);
        }
      };
    };

    connect();

    return () => {
      cancelled = true;
      clearReconnect();
      stopWatch();
      wsRef.current?.close();
    };
  }, [enabled, startLocationStream, stopWatch, token]);

  return (
    <View style={styles.stack}>
      <ScreenHeading
        title="Safety GPS Channel"
        subtitle="WS connection stands ready for location tracing during emergency calls."
      />
      <View style={styles.card}>
        <View style={styles.rowBetween}>
          <View style={styles.flex}>
            <Text style={styles.cardTitle}>
              {connected ? "Connection Secure" : "Channel Offline"}
            </Text>
            <Text style={styles.cardBody}>{status}</Text>
          </View>
          <Ionicons
            name={connected ? "radio-outline" : "radio-button-off-outline"}
            size={26}
            color={connected ? "#059669" : "#64748b"}
          />
        </View>
        <Text style={styles.meta}>
          Session ID Ref: {activeSession || "None active"}
        </Text>
      </View>
    </View>
  );
}

function PatientWelfareChecks({
  checks,
  scheduledAt,
  setScheduledAt,
  reason,
  setReason,
  detail,
  setDetail,
  timezone,
  creating,
  cancellingId,
  onCreate,
  onCancel,
  onRefresh,
}: {
  checks: PatientWelfareCheck[];
  scheduledAt: string;
  setScheduledAt: (v: string) => void;
  reason: WelfareCheckReason;
  setReason: (v: WelfareCheckReason) => void;
  detail: string;
  setDetail: (v: string) => void;
  timezone: string;
  creating: boolean;
  cancellingId: string | null;
  onCreate: () => void;
  onCancel: (id: string) => void;
  onRefresh: () => void;
}) {
  return (
    <View style={styles.stack}>
      <ScreenHeading
        title="Welfare Checks"
        subtitle="Schedule an outbound Zorba AI phone check with the reason and details you choose."
      />
      <Section title="Schedule a check">
        <Field
          label={`When (${timezone})`}
          value={scheduledAt}
          onChangeText={setScheduledAt}
          placeholder="YYYY-MM-DDTHH:MM"
          autoCapitalize="none"
        />
        <Text style={styles.meta}>Reason</Text>
        <View style={styles.inlineActions}>
          {welfareReasons.map((code) => (
            <IconButton
              key={code}
              icon={reason === code ? "checkmark-circle" : "ellipse-outline"}
              label={welfareReasonLabels[code]}
              onPress={() => setReason(code)}
              tone={reason === code ? "accent" : "neutral"}
            />
          ))}
        </View>
        <Field
          label="Details (optional)"
          value={detail}
          onChangeText={setDetail}
          placeholder="Anything Zorba should mention on the call"
        />
        <PrimaryButton
          icon="heart-outline"
          label={creating ? "Scheduling..." : "Schedule welfare check"}
          disabled={creating}
          onPress={onCreate}
        />
      </Section>
      <Section title="Upcoming & recent">
        <IconButton icon="refresh-outline" label="Refresh" onPress={onRefresh} tone="neutral" />
        {checks.length === 0 ? (
          <EmptyText text="No welfare checks scheduled yet." />
        ) : (
          checks.map((check) => {
            const cancelled = check.status === "cancelled";
            const reasonLabel =
              welfareReasonLabels[(check.reason_code as WelfareCheckReason) || "other"] ||
              check.reason_code ||
              "Welfare check";
            return (
              <View key={check.id} style={styles.card}>
                <View style={styles.rowBetween}>
                  <View style={styles.flex}>
                    <Text style={styles.cardTitle}>{reasonLabel}</Text>
                    <Text style={styles.meta}>{formatTime(check.scheduled_at)}</Text>
                    {check.reason_detail ? (
                      <Text style={styles.cardBody}>{check.reason_detail}</Text>
                    ) : null}
                  </View>
                  <Text style={[styles.badge, cancelled ? styles.badgeWarn : null]}>
                    {check.status || "scheduled"}
                  </Text>
                </View>
                {!cancelled && check.status !== "completed" && check.status !== "missed" ? (
                  <IconButton
                    icon="close-outline"
                    label={cancellingId === check.id ? "Cancelling..." : "Cancel"}
                    onPress={() => onCancel(check.id)}
                    tone="neutral"
                  />
                ) : null}
              </View>
            );
          })
        )}
      </Section>
    </View>
  );
}

function PatientMeetings({
  token,
  meetings,
  setMeetings,
  schedulableStaff,
  hospitalConsents,
  showScheduleForm,
  setShowScheduleForm,
  scheduleStaffID,
  setScheduleStaffID,
  scheduleHospitalID,
  setScheduleHospitalID,
  scheduleStartsAt,
  setScheduleStartsAt,
  scheduleDuration,
  setScheduleDuration,
  scheduleTimezone,
  setScheduleTimezone,
  scheduleTitle,
  setScheduleTitle,
  scheduleNotes,
  setScheduleNotes,
  submittingSchedule,
  mutatingMeetingID,
  onSchedule,
  onCancel,
  onLoadSchedulableStaff,
  onRefresh,
}: {
  token: string;
  meetings: HospitalMeeting[];
  setMeetings: React.Dispatch<React.SetStateAction<HospitalMeeting[]>>;
  schedulableStaff: PatientSchedulableStaffMember[];
  hospitalConsents: PatientHospitalConsent[];
  showScheduleForm: boolean;
  setShowScheduleForm: (v: boolean) => void;
  scheduleStaffID: string;
  setScheduleStaffID: (v: string) => void;
  scheduleHospitalID: string;
  setScheduleHospitalID: (v: string) => void;
  scheduleStartsAt: string;
  setScheduleStartsAt: (v: string) => void;
  scheduleDuration: string;
  setScheduleDuration: (v: string) => void;
  scheduleTimezone: string;
  setScheduleTimezone: (v: string) => void;
  scheduleTitle: string;
  setScheduleTitle: (v: string) => void;
  scheduleNotes: string;
  setScheduleNotes: (v: string) => void;
  submittingSchedule: boolean;
  mutatingMeetingID: string;
  onSchedule: () => void;
  onCancel: (id: string) => void;
  onLoadSchedulableStaff: (hospitalID: string) => void;
  onRefresh: () => void;
}) {
  const [error, setError] = useState("");
  const [activeJoinURL, setActiveJoinURL] = useState<string | null>(null);
  const activeConsents = useMemo(
    () => hospitalConsents.filter((hc) => !hc.revoked_at),
    [hospitalConsents],
  );

  if (activeJoinURL) {
    return (
      <MeetingVideoRoom joinURL={activeJoinURL} onClose={() => setActiveJoinURL(null)} />
    );
  }

  return (
    <View style={styles.stack}>
      <ScreenHeading
        title="My Meetings"
        subtitle="Schedule video visits with hospital staff."
      />
      <View style={{ flexDirection: "row", gap: 8, flexWrap: "wrap" }}>
        <IconButton
          icon={showScheduleForm ? "close-outline" : "add-outline"}
          label={showScheduleForm ? "Back to List" : "Schedule New"}
          onPress={() => {
            if (!showScheduleForm && activeConsents.length > 0) {
              const first = activeConsents[0];
              setScheduleHospitalID(first.hospital_id || "");
              onLoadSchedulableStaff(first.hospital_id || "");
            }
            setShowScheduleForm(!showScheduleForm);
          }}
          tone="accent"
        />
        <IconButton
          icon="refresh-outline"
          label="Refresh"
          onPress={onRefresh}
          tone="neutral"
        />
      </View>
      <Feedback error={error} />

      {showScheduleForm ? (
        <View style={styles.card}>
          <Text style={styles.sectionTitle}>Schedule New Meeting</Text>
          {activeConsents.length > 0 ? (
            <View style={{ gap: 4, marginBottom: 8 }}>
              <Text style={styles.label}>Hospital</Text>
              {activeConsents.map((hc) => {
                const selected = scheduleHospitalID === hc.hospital_id;
                return (
                  <Pressable
                    key={hc.hospital_id}
                    onPress={() => {
                      setScheduleHospitalID(hc.hospital_id || "");
                      onLoadSchedulableStaff(hc.hospital_id || "");
                    }}
                    style={[
                      styles.card,
                      selected && {
                        borderColor: "#4f46e5",
                        borderWidth: 2,
                      },
                    ]}
                  >
                    <Text style={styles.cardTitle}>{hc.hospital_name || hc.hospital_id}</Text>
                    <Text style={styles.meta}>{hc.status || "active"}</Text>
                  </Pressable>
                );
              })}
            </View>
          ) : (
            <Field
              label="Hospital ID"
              value={scheduleHospitalID}
              onChangeText={(value) => {
                setScheduleHospitalID(value);
                onLoadSchedulableStaff(value);
              }}
              placeholder="Hospital ID"
            />
          )}
          {schedulableStaff.length > 0 ? (
            <View style={{ gap: 4, marginBottom: 8 }}>
              <Text style={styles.label}>Available Staff</Text>
              {schedulableStaff.map((staff) => {
                const selected = scheduleStaffID === staff.staff_id;
                return (
                  <Pressable
                    key={staff.staff_id}
                    onPress={() => setScheduleStaffID(staff.staff_id)}
                    style={[
                      styles.card,
                      selected && {
                        borderColor: "#4f46e5",
                        borderWidth: 2,
                      },
                    ]}
                  >
                    <Text style={styles.cardTitle}>{staff.name || staff.staff_id}</Text>
                    <Text style={styles.meta}>
                      {staff.role || "staff"} • {staff.email}
                    </Text>
                  </Pressable>
                );
              })}
            </View>
          ) : scheduleHospitalID ? (
            <Text style={styles.muted}>No staff available for this hospital.</Text>
          ) : null}
          <Field
            label="Date & Time"
            value={scheduleStartsAt}
            onChangeText={setScheduleStartsAt}
            placeholder="e.g. 2026-06-10T14:00"
          />
          <Field
            label="Duration (minutes)"
            value={scheduleDuration}
            onChangeText={setScheduleDuration}
            placeholder="30"
            keyboardType="number-pad"
          />
          <Field
            label="Timezone (IANA)"
            value={scheduleTimezone}
            onChangeText={setScheduleTimezone}
            placeholder="America/Chicago"
            autoCapitalize="none"
          />
          <Field
            label="Title"
            value={scheduleTitle}
            onChangeText={setScheduleTitle}
            placeholder="Zorba Health video visit"
          />
          <Field
            label="Notes"
            value={scheduleNotes}
            onChangeText={setScheduleNotes}
            placeholder="Optional notes"
            multiline
          />
          <PrimaryButton
            icon="calendar-outline"
            label={submittingSchedule ? "Scheduling..." : "Schedule Meeting"}
            onPress={onSchedule}
            disabled={submittingSchedule}
          />
        </View>
      ) : null}

      {!showScheduleForm && meetings.length === 0 ? (
        <EmptyText text="No meetings scheduled." />
      ) : null}

      {!showScheduleForm
        ? meetings.map((meeting) => {
            const cancelled = meeting.status === "cancelled";
            return (
              <View key={meeting.id} style={styles.card}>
                <View style={styles.rowBetween}>
                  <View style={styles.flex}>
                    <Text style={styles.cardTitle}>
                      {meeting.title || "Zorba Health video visit"}
                    </Text>
                    <Text style={styles.cardBody}>
                      Staff Ref: {meeting.staff_id || "Unassigned"}
                    </Text>
                    <Text style={styles.meta}>
                      {formatTime(meeting.starts_at)} • {meeting.duration_minutes || 30} min
                    </Text>
                  </View>
                  <Text style={[styles.badge, cancelled ? styles.badgeWarn : null]}>
                    {meeting.status || "pending"}
                  </Text>
                </View>
                <View style={styles.inlineActions}>
                  {meeting.join_url && !cancelled ? (
                    <IconButton
                      icon="videocam-outline"
                      label="Join"
                      onPress={() => setActiveJoinURL(meeting.join_url as string)}
                      tone="accent"
                    />
                  ) : null}
                  {!cancelled ? (
                    <IconButton
                      icon="close-outline"
                      label={
                        mutatingMeetingID === meeting.id ? "Cancelling" : "Cancel"
                      }
                      onPress={() => onCancel(meeting.id)}
                      tone="neutral"
                    />
                  ) : null}
                </View>
              </View>
            );
          })
        : null}
    </View>
  );
}

function HospitalPortal({
  token,
  onSignOut,
}: {
  token: string;
  onSignOut: () => void;
}) {
  const [tab, setTab] = useState<HospitalTab>("home");
  const [patients, setPatients] = useState<HospitalPatient[]>([]);
  const [patientLookup, setPatientLookup] = useState("");
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [meetings, setMeetings] = useState<HospitalMeeting[]>([]);
  const [consentRequests, setConsentRequests] = useState<HospitalConsentRequest[]>([]);
  const [loadingIncidents, setLoadingIncidents] = useState(false);
  const [loadingPatients, setLoadingPatients] = useState(false);
  const [loadingMeetings, setLoadingMeetings] = useState(false);
  const [loadingConsentRequests, setLoadingConsentRequests] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [summaryRefreshSignal, setSummaryRefreshSignal] = useState(0);
  const [auditRefreshSignal, setAuditRefreshSignal] = useState(0);
  const [showHospitalSchedule, setShowHospitalSchedule] = useState(false);
  const [hospSchedulePatientID, setHospSchedulePatientID] = useState("");
  const [hospScheduleStartsAt, setHospScheduleStartsAt] = useState("");
  const [hospScheduleDuration, setHospScheduleDuration] = useState("30");
  const [hospScheduleTimezone, setHospScheduleTimezone] = useState(() =>
    resolveIANATimezone(),
  );
  const [hospScheduleTitle, setHospScheduleTitle] = useState("");
  const [hospScheduleNotes, setHospScheduleNotes] = useState("");
  const [submittingHospSchedule, setSubmittingHospSchedule] = useState(false);

  const loadPatients = useCallback(
    async (query = "") => {
      setLoadingPatients(true);
      try {
        const suffix = query.trim()
          ? `?query=${encodeURIComponent(query.trim())}`
          : "";
        const data = await apiRequest<{ patients?: HospitalPatient[] }>(
          `${endpoints.hospitalPatients}${suffix}`,
          { token, role: "hospital" },
        );
        setPatients(data.patients ?? []);
      } catch (err) {
        if (err instanceof Error && err.message.toLowerCase().includes("token"))
          onSignOut();
      } finally {
        setLoadingPatients(false);
      }
    },
    [onSignOut, token],
  );

  const loadIncidents = useCallback(async () => {
    setLoadingIncidents(true);
    try {
      const data = await apiRequest<{ incidents?: Incident[] }>(
        endpoints.hospitalIncidents,
        { token },
      );
      setIncidents(data.incidents ?? []);
    } catch (err) {
      if (err instanceof Error && err.message.toLowerCase().includes("token"))
        onSignOut();
    } finally {
      setLoadingIncidents(false);
    }
  }, [onSignOut, token]);

  const loadMeetings = useCallback(async () => {
    setLoadingMeetings(true);
    try {
      const data = await apiRequest<{ meetings?: HospitalMeeting[] }>(
        endpoints.hospitalMeetings,
        { token, role: "hospital" },
      );
      setMeetings(data.meetings ?? []);
    } catch (err) {
      if (err instanceof Error && err.message.toLowerCase().includes("token"))
        onSignOut();
    } finally {
      setLoadingMeetings(false);
    }
  }, [onSignOut, token]);

  const loadConsentRequests = useCallback(async () => {
    setLoadingConsentRequests(true);
    try {
      const data = await apiRequest<{ requests?: HospitalConsentRequest[] }>(
        endpoints.hospitalConsentRequests,
        { token, role: "hospital" },
      );
      setConsentRequests(data.requests ?? []);
    } catch (err) {
      if (err instanceof Error && err.message.toLowerCase().includes("token"))
        onSignOut();
    } finally {
      setLoadingConsentRequests(false);
    }
  }, [onSignOut, token]);

  const handleHospitalSchedule = async () => {
    const timezone = resolveIANATimezone(hospScheduleTimezone);
    if (timezone !== hospScheduleTimezone) {
      setHospScheduleTimezone(timezone);
    }
    setSubmittingHospSchedule(true);
    try {
      const data = await apiRequest<{ meeting?: HospitalMeeting }>(
        endpoints.hospitalMeetings,
        {
          method: "POST",
          token,
          role: "hospital",
          body: {
            patient_id: hospSchedulePatientID.trim(),
            starts_at: new Date(hospScheduleStartsAt).toISOString(),
            duration_minutes: Number(hospScheduleDuration) || 30,
            timezone,
            title: hospScheduleTitle || undefined,
            notes: hospScheduleNotes || undefined,
          },
        },
      );
      if (data.meeting) setMeetings((prev) => [data.meeting!, ...prev]);
      setShowHospitalSchedule(false);
      setHospSchedulePatientID("");
      setHospScheduleStartsAt("");
      setHospScheduleTitle("");
      setHospScheduleNotes("");
    } catch (err) {
      Alert.alert("Schedule failed", err instanceof Error ? err.message : "Unknown error");
    } finally {
      setSubmittingHospSchedule(false);
    }
  };

  useEffect(() => {
    void loadPatients();
    void loadIncidents();
    void loadMeetings();
    void loadConsentRequests();
  }, [loadConsentRequests, loadIncidents, loadMeetings, loadPatients]);

  const completeRefresh = useCallback(() => {
    setRefreshing(false);
  }, []);

  const refreshCurrentTab = useCallback(() => {
    setRefreshing(true);
    if (tab === "incidents") {
      void loadIncidents().finally(completeRefresh);
      return;
    }
    if (tab === "home") {
      void loadPatients().finally(completeRefresh);
      return;
    }
    if (tab === "meetings") {
      void loadMeetings().finally(completeRefresh);
      return;
    }
    if (tab === "consent") {
      void loadConsentRequests().finally(completeRefresh);
      return;
    }
    if (tab === "summary") {
      setSummaryRefreshSignal((current) => current + 1);
      return;
    }
    if (tab === "audit") {
      setAuditRefreshSignal((current) => current + 1);
      return;
    }
    setRefreshing(false);
  }, [completeRefresh, loadConsentRequests, loadIncidents, loadMeetings, loadPatients, tab]);

  return (
    <View style={styles.portal}>
      <TabBar
        value={tab}
        onChange={(value) => setTab(value as HospitalTab)}
        options={[
          { value: "home", label: "Home", icon: "home-outline" },
          { value: "summary", label: "Summarize", icon: "medkit-outline" },
          { value: "meetings", label: "Meetings", icon: "calendar-outline" },
          { value: "staff", label: "Staff", icon: "person-add-outline" },
          { value: "consent", label: "Consent", icon: "qr-code-outline" },
          {
            value: "incidents",
            label: "Alerts Inbox",
            icon: "alert-circle-outline",
          },
          { value: "audit", label: "Audit Search", icon: "reader-outline" },
        ]}
      />
      <ScrollView
        contentContainerStyle={styles.content}
        showsVerticalScrollIndicator={false}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={refreshCurrentTab}
            tintColor="#4f46e5"
            colors={["#4f46e5"]}
          />
        }
      >
        {tab === "home" ? (
          <View style={styles.stack}>
            <HospitalIncomingBridges token={token} />
            <HospitalHome
              patients={patients}
              loading={loadingPatients}
              onSearch={loadPatients}
              onSummary={(lookup) => {
                setPatientLookup(lookup);
                setTab("summary");
              }}
              onAudit={(lookup) => {
                setPatientLookup(lookup);
                setTab("audit");
              }}
            />
          </View>
        ) : null}
        {tab === "summary" ? (
          <HospitalSummary
            token={token}
            initialLookup={patientLookup}
            refreshSignal={summaryRefreshSignal}
            onRefreshComplete={completeRefresh}
          />
        ) : null}
        {tab === "incidents" ? (
          <IncidentList
            incidents={incidents}
            loading={loadingIncidents}
            onRefresh={loadIncidents}
          />
        ) : null}
        {tab === "meetings" ? (
          <HospitalMeetings
            token={token}
            meetings={meetings}
            loading={loadingMeetings}
            setMeetings={setMeetings}
            onRefresh={loadMeetings}
            showScheduleForm={showHospitalSchedule}
            setShowScheduleForm={setShowHospitalSchedule}
            schedulePatientID={hospSchedulePatientID}
            setSchedulePatientID={setHospSchedulePatientID}
            scheduleStartsAt={hospScheduleStartsAt}
            setScheduleStartsAt={setHospScheduleStartsAt}
            scheduleDuration={hospScheduleDuration}
            setScheduleDuration={setHospScheduleDuration}
            scheduleTimezone={hospScheduleTimezone}
            setScheduleTimezone={setHospScheduleTimezone}
            scheduleTitle={hospScheduleTitle}
            setScheduleTitle={setHospScheduleTitle}
            scheduleNotes={hospScheduleNotes}
            setScheduleNotes={setHospScheduleNotes}
            submittingSchedule={submittingHospSchedule}
            onSchedule={handleHospitalSchedule}
          />
        ) : null}
        {tab === "staff" ? <HospitalStaffRegistration token={token} /> : null}
        {tab === "consent" ? (
          <HospitalConsentRequests
            token={token}
            requests={consentRequests}
            loading={loadingConsentRequests}
            setRequests={setConsentRequests}
            onRefresh={loadConsentRequests}
          />
        ) : null}
        {tab === "audit" ? (
          <HospitalAudit
            token={token}
            initialLookup={patientLookup}
            refreshSignal={auditRefreshSignal}
            onRefreshComplete={completeRefresh}
          />
        ) : null}
      </ScrollView>
    </View>
  );
}

function HospitalIncomingBridges({ token }: { token: string }) {
  const [pending, setPending] = useState<BridgedCallSession[]>([]);
  const [languageCode, setLanguageCode] = useState("en");
  const [busySession, setBusySession] = useState<string | null>(null);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [bridgeRoomState, setBridgeRoomState] = useState<"disconnected" | "connecting" | "connected">("disconnected");
  const bridgeRoomRef = useRef<Room | null>(null);

  const disconnectBridgeRoom = useCallback(async () => {
    const room = bridgeRoomRef.current;
    bridgeRoomRef.current = null;
    setBridgeRoomState("disconnected");
    if (room) {
      try {
        await room.disconnect();
      } catch {
        // Best-effort teardown.
      }
    }
  }, []);

  useEffect(() => {
    return () => {
      void disconnectBridgeRoom();
    };
  }, [disconnectBridgeRoom]);

  const loadPending = useCallback(async () => {
    try {
      const data = await apiRequest<{ sessions?: BridgedCallSession[] }>(
        `${endpoints.hospitalBridgedCallSessions}?status=transfer_requested`,
        { token, role: "hospital" },
      );
      setPending(data.sessions ?? []);
    } catch {
      // Best-effort polling.
    }
  }, [token]);

  useEffect(() => {
    void loadPending();
    const timer = setInterval(() => {
      void loadPending();
    }, 8000);
    return () => clearInterval(timer);
  }, [loadPending]);

  const joinBridgeRoom = useCallback(
    async (wsUrl: string, roomToken: string) => {
      await disconnectBridgeRoom();
      const room = new Room();
      bridgeRoomRef.current = room;
      setBridgeRoomState("connecting");
      room.on(RoomEvent.Disconnected, () => {
        if (bridgeRoomRef.current === room) {
          bridgeRoomRef.current = null;
          setBridgeRoomState("disconnected");
        }
      });
      try {
        await room.connect(wsUrl, roomToken);
        await room.localParticipant.setMicrophoneEnabled(true).catch(() => undefined);
        setBridgeRoomState("connected");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unable to join LiveKit room.");
        await disconnectBridgeRoom();
      }
    },
    [disconnectBridgeRoom],
  );

  const acceptCall = useCallback(
    async (sessionId: string, joinMode: "web" | "phone") => {
      setBusySession(sessionId);
      setError("");
      setNotice("");
      try {
        await apiRequest<{ session?: BridgedCallSession }>(
          endpoints.hospitalBridgedCallTranslation,
          {
            method: "PUT",
            token,
            role: "hospital",
            body: {
              session_id: sessionId,
              participant: "staff",
              translation: {
                enabled: true,
                language_mode: "manual",
                language_code: languageCode.trim() || "en",
              },
            },
          },
        ).catch(() => undefined);
        const data = await apiRequest<BridgedCallSessionResponseData>(
          endpoints.hospitalBridgedCallConnect,
          {
            method: "POST",
            token,
            role: "hospital",
            body: {
              session_id: sessionId,
              join_mode: joinMode,
            },
          },
        );
        setNotice(
          joinMode === "phone"
            ? "Dialing your staff phone into the patient call..."
            : "Joined the bridged call in-app.",
        );
        void loadPending();
        if (joinMode === "web" && data.staff_room_token && data.livekit_ws_url) {
          await joinBridgeRoom(data.livekit_ws_url, data.staff_room_token);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unable to accept bridged call.");
      } finally {
        setBusySession(null);
      }
    },
    [joinBridgeRoom, languageCode, loadPending, token],
  );

  return (
    <Section title="Incoming bridged calls">
      <Text style={styles.cardBody}>
        When a patient presses 0 or requests staff during a Zorba call, accept here. Choose phone to dial your staff line via LiveKit SIP, or web to join in-app with interpretation.
      </Text>
      <Field
        label="Your language"
        value={languageCode}
        onChangeText={setLanguageCode}
        autoCapitalize="none"
        placeholder="en"
      />
      <Feedback error={error} notice={notice} />
      {bridgeRoomState !== "disconnected" ? (
        <Text style={styles.meta}>Live room: {bridgeRoomState}</Text>
      ) : null}
      {pending.length === 0 ? (
        <EmptyText text="No patients waiting for a bridged consult." />
      ) : (
        pending.map((session) => (
          <View key={session.session_id} style={[styles.card, { borderColor: "#f59e0b", borderWidth: 2 }]}>
            <Text style={styles.badge}>Ringing</Text>
            <Text style={styles.cardTitle}>{session.session_id}</Text>
            <Text style={styles.cardBody}>
              Patient {session.patient_id || "unknown"}
              {session.transfer_reason ? ` — ${session.transfer_reason}` : ""}
            </Text>
            <View style={styles.inlineActions}>
              <IconButton
                icon="laptop-outline"
                label={busySession === session.session_id ? "Connecting..." : "Accept (Web)"}
                onPress={() => void acceptCall(session.session_id, "web")}
                tone="accent"
              />
              <IconButton
                icon="call-outline"
                label="Accept (Phone)"
                onPress={() => void acceptCall(session.session_id, "phone")}
                tone="neutral"
              />
            </View>
          </View>
        ))
      )}
      <IconButton icon="refresh-outline" label="Refresh" onPress={() => void loadPending()} tone="neutral" />
    </Section>
  );
}

function HospitalHome({
  patients,
  loading,
  onSearch,
  onSummary,
  onAudit,
}: {
  patients: HospitalPatient[];
  loading: boolean;
  onSearch: (query?: string) => void;
  onSummary: (lookup: string) => void;
  onAudit: (lookup: string) => void;
}) {
  const [query, setQuery] = useState("");

  return (
    <View style={styles.stack}>
      <ScreenHeading
        title="Consented Patients"
        subtitle="Patients with active consent for this hospital."
      />
      <Field
        label="Patient Lookup"
        value={query}
        onChangeText={setQuery}
        autoCapitalize="none"
        placeholder="Name, email, or patient ID"
      />
      <View style={styles.inlineActions}>
        <IconButton
          icon="search-outline"
          label={loading ? "Searching..." : "Search"}
          onPress={() => onSearch(query)}
          tone="accent"
        />
        <IconButton
          icon="refresh-outline"
          label="All patients"
          onPress={() => {
            setQuery("");
            onSearch("");
          }}
          tone="neutral"
        />
      </View>
      {patients.map((patient) => {
        const lookup = patient.patient_id || patient.email || patient.full_name || "";
        return (
          <View key={lookup} style={styles.card}>
            <View style={styles.rowBetween}>
              <View style={styles.flex}>
                <Text style={styles.cardTitle}>
                  {patient.full_name || "Unnamed patient"}
                </Text>
                <Text style={styles.cardBody}>
                  {patient.email || "No email on file"}
                </Text>
              </View>
              <Text style={styles.badge}>Active</Text>
            </View>
            <Text style={styles.meta}>ID: {patient.patient_id || "Unknown"}</Text>
            <Text style={styles.meta}>
              Consent: {formatTime(patient.consent_granted_at)}
            </Text>
            <Text style={styles.meta}>
              Last call: {patient.last_call_at ? formatTime(patient.last_call_at) : "No calls yet"}
            </Text>
            <View style={styles.inlineActions}>
              <IconButton
                icon="sparkles-outline"
                label="Summary"
                onPress={() => onSummary(lookup)}
                tone="accent"
              />
              <IconButton
                icon="reader-outline"
                label="Audit"
                onPress={() => onAudit(lookup)}
                tone="neutral"
              />
            </View>
          </View>
        );
      })}
      {patients.length === 0 ? (
        <EmptyText
          text={loading ? "Loading consented patients..." : "No active hospital consents matched."}
        />
      ) : null}
    </View>
  );
}

function HospitalSummary({
  token,
  initialLookup = "",
  refreshSignal = 0,
  onRefreshComplete,
}: {
  token: string;
  initialLookup?: string;
  refreshSignal?: number;
  onRefreshComplete?: () => void;
}) {
  const [patientID, setPatientID] = useState(initialLookup);
  const [focus, setFocus] = useState("full");
  const [summary, setSummary] = useState("");
  const [audit, setAudit] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    setPatientID(initialLookup);
  }, [initialLookup]);

  const generate = useCallback(async () => {
    if (!patientID.trim()) return;
    setLoading(true);
    setError("");
    setSummary("");
    setAudit([]);
    try {
      const data = await apiRequest<{ patient_id?: string; summary?: string }>(
        endpoints.hospitalSummary,
        {
          method: "POST",
          token,
          body: { patient_id: patientID.trim(), focus },
        },
      );
      if (data.patient_id) setPatientID(data.patient_id);
      setSummary(data.summary ?? "No summary returned.");
      const auditData = await apiRequest<{ events?: AuditEvent[] }>(
        `${endpoints.hospitalPatientAudit}?patient_id=${encodeURIComponent(data.patient_id || patientID.trim())}`,
        { token },
      );
      setAudit(auditData.events ?? []);
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Unable to summarize patient records.",
      );
    } finally {
      setLoading(false);
    }
  }, [focus, patientID, token]);

  useEffect(() => {
    if (!refreshSignal) return;
    void generate().finally(onRefreshComplete);
  }, [generate, onRefreshComplete, refreshSignal]);

  return (
    <View style={styles.stack}>
      <ScreenHeading
        title="Record Summarizer"
        subtitle="Generate clinical record translations based on compliant database traces."
      />
      <Field
        label="Target Patient"
        value={patientID}
        onChangeText={setPatientID}
        autoCapitalize="none"
        placeholder="Name, email, or patient ID"
      />
      <Text style={styles.label}>Focus Area Select</Text>
      <Segmented
        value={focus}
        options={focusOptions.map((value) => ({ value, label: value }))}
        onChange={setFocus}
      />
      <PrimaryButton
        icon="sparkles-outline"
        label={loading ? "Compiling..." : "Generate Summary"}
        disabled={loading}
        onPress={generate}
      />
      <Feedback error={error} />

      {summary ? (
        <Section title="Generated Summary Document">
          <View style={styles.card}>
            <Text style={styles.bodyText}>{summary}</Text>
          </View>
        </Section>
      ) : null}

      {audit.length > 0 ? (
        <Section title="Patient Audit Access Ledger">
          <AuditList events={audit} compact />
        </Section>
      ) : null}
    </View>
  );
}

function HospitalMeetings({
  token,
  meetings,
  loading,
  setMeetings,
  onRefresh,
  showScheduleForm,
  setShowScheduleForm,
  schedulePatientID,
  setSchedulePatientID,
  scheduleStartsAt,
  setScheduleStartsAt,
  scheduleDuration,
  setScheduleDuration,
  scheduleTimezone,
  setScheduleTimezone,
  scheduleTitle,
  setScheduleTitle,
  scheduleNotes,
  setScheduleNotes,
  submittingSchedule,
  onSchedule,
}: {
  token: string;
  meetings: HospitalMeeting[];
  loading: boolean;
  setMeetings: React.Dispatch<React.SetStateAction<HospitalMeeting[]>>;
  onRefresh: () => void;
  showScheduleForm?: boolean;
  setShowScheduleForm?: (v: boolean) => void;
  schedulePatientID?: string;
  setSchedulePatientID?: (v: string) => void;
  scheduleStartsAt?: string;
  setScheduleStartsAt?: (v: string) => void;
  scheduleDuration?: string;
  setScheduleDuration?: (v: string) => void;
  scheduleTimezone?: string;
  setScheduleTimezone?: (v: string) => void;
  scheduleTitle?: string;
  setScheduleTitle?: (v: string) => void;
  scheduleNotes?: string;
  setScheduleNotes?: (v: string) => void;
  submittingSchedule?: boolean;
  onSchedule?: () => void;
}) {
  const [mutating, setMutating] = useState("");
  const [error, setError] = useState("");
  const [activeJoinURL, setActiveJoinURL] = useState<string | null>(null);

  const mergeMeeting = (meeting?: HospitalMeeting) => {
    if (!meeting?.id) return;
    setMeetings((current) =>
      current.some((item) => item.id === meeting.id)
        ? current.map((item) => (item.id === meeting.id ? meeting : item))
        : [meeting, ...current],
    );
  };

  const mutate = async (meeting: HospitalMeeting, action: "accept" | "cancel") => {
    if (!meeting.id) return;
    setMutating(meeting.id);
    setError("");
    try {
      const path =
        action === "accept"
          ? `${endpoints.hospitalMeetings}/${encodeURIComponent(meeting.id)}/accept`
          : `${endpoints.hospitalMeetings}/${encodeURIComponent(meeting.id)}`;
      const data = await apiRequest<{ meeting?: HospitalMeeting }>(path, {
        method: action === "accept" ? "POST" : "DELETE",
        token,
        role: "hospital",
        body:
          action === "cancel"
            ? { reason: "Cancelled from mobile hospital console" }
            : undefined,
      });
      mergeMeeting(data.meeting);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Meeting update failed.");
    } finally {
      setMutating("");
    }
  };

  if (activeJoinURL) {
    return (
      <MeetingVideoRoom joinURL={activeJoinURL} onClose={() => setActiveJoinURL(null)} />
    );
  }

  return (
    <View style={styles.stack}>
      <ScreenHeading
        title="Visit Requests"
        subtitle="Review patient scheduling requests and accept or cancel from mobile."
      />
      <View style={{ flexDirection: "row", gap: 8, flexWrap: "wrap" }}>
        <IconButton
          icon={showScheduleForm ? "close-outline" : "add-outline"}
          label={showScheduleForm ? "Back to List" : "Schedule New"}
          onPress={() => setShowScheduleForm?.(!showScheduleForm)}
          tone="accent"
        />
        <IconButton
          icon="refresh-outline"
          label={loading ? "Refreshing..." : "Refresh"}
          onPress={onRefresh}
          tone="neutral"
        />
      </View>
      <Feedback error={error} />

      {showScheduleForm && onSchedule ? (
        <View style={styles.card}>
          <Text style={styles.sectionTitle}>Schedule New Meeting</Text>
          <Field
            label="Patient ID"
            value={schedulePatientID || ""}
            onChangeText={(v) => setSchedulePatientID?.(v)}
            placeholder="Enter patient ID"
          />
          <Field
            label="Date & Time"
            value={scheduleStartsAt || ""}
            onChangeText={(v) => setScheduleStartsAt?.(v)}
            placeholder="e.g. 2026-06-10T14:00"
          />
          <Field
            label="Duration (minutes)"
            value={scheduleDuration || "30"}
            onChangeText={(v) => setScheduleDuration?.(v)}
            keyboardType="number-pad"
          />
          <Field
            label="Timezone (IANA)"
            value={scheduleTimezone || "UTC"}
            onChangeText={(v) => setScheduleTimezone?.(v)}
            placeholder="America/Chicago"
            autoCapitalize="none"
          />
          <Field
            label="Title"
            value={scheduleTitle || ""}
            onChangeText={(v) => setScheduleTitle?.(v)}
            placeholder="Zorba Health video visit"
          />
          <Field
            label="Notes"
            value={scheduleNotes || ""}
            onChangeText={(v) => setScheduleNotes?.(v)}
            placeholder="Optional notes"
            multiline
          />
          <PrimaryButton
            icon="calendar-outline"
            label={submittingSchedule ? "Scheduling..." : "Schedule Meeting"}
            onPress={onSchedule}
            disabled={submittingSchedule}
          />
        </View>
      ) : null}

      {!showScheduleForm && meetings.length === 0 ? (
        <EmptyText text={loading ? "Loading meetings..." : "No visit requests found."} />
      ) : null}

      {!showScheduleForm
        ? meetings.map((meeting) => {
            const pending = meeting.status === "pending";
            const cancelled = meeting.status === "cancelled";
            return (
              <View key={meeting.id} style={styles.card}>
                <View style={styles.rowBetween}>
                  <View style={styles.flex}>
                    <Text style={styles.cardTitle}>
                      {meeting.title || "Zorba Health video visit"}
                    </Text>
                    <Text style={styles.cardBody}>
                      Patient Ref: {meeting.patient_id || "Unlisted"}
                    </Text>
                    <Text style={styles.meta}>
                      {formatTime(meeting.starts_at)} • {meeting.duration_minutes || 30} min
                    </Text>
                  </View>
                  <Text style={[styles.badge, cancelled ? styles.badgeWarn : null]}>
                    {meeting.status || "pending"}
                  </Text>
                </View>
                <View style={styles.inlineActions}>
                  {pending ? (
                    <IconButton
                      icon="checkmark-outline"
                      label={mutating === meeting.id ? "Accepting" : "Accept"}
                      onPress={() => mutate(meeting, "accept")}
                      tone="accent"
                    />
                  ) : null}
                  {meeting.join_url ? (
                    <IconButton
                      icon="videocam-outline"
                      label="Join"
                      onPress={() => setActiveJoinURL(meeting.join_url as string)}
                      tone="neutral"
                    />
                  ) : null}
                  {!cancelled ? (
                    <IconButton
                      icon="close-outline"
                      label={mutating === meeting.id ? "Cancelling" : "Cancel"}
                      onPress={() => mutate(meeting, "cancel")}
                      tone="neutral"
                    />
                  ) : null}
                </View>
              </View>
            );
          })
        : null}
    </View>
  );
}

function HospitalStaffRegistration({ token }: { token: string }) {
  const [staffName, setStaffName] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState("doctor");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const submit = async () => {
    setSubmitting(true);
    setError("");
    setNotice("");
    try {
      const data = await apiRequest<{ staff_id?: string; message?: string }>(
        endpoints.hospitalStaffRegister,
        {
          method: "POST",
          token,
          role: "hospital",
          body: {
            staff_name: staffName.trim(),
            email: email.trim(),
            phone_number: phone.trim() || undefined,
            password,
            staff_role: role,
          },
        },
      );
      setNotice(data.message || `Staff account created${data.staff_id ? `: ${data.staff_id}` : "."}`);
      setStaffName("");
      setEmail("");
      setPhone("");
      setPassword("");
      setRole("doctor");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Staff registration failed.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <View style={styles.stack}>
      <ScreenHeading
        title="Register Staff"
        subtitle="Create hospital staff login credentials linked to this hospital."
      />
      <Feedback error={error} notice={notice} />
      <Field label="Staff Name" value={staffName} onChangeText={setStaffName} placeholder="Dr. Jane Clinician" />
      <Field label="Email" value={email} onChangeText={setEmail} keyboardType="email-address" autoCapitalize="none" placeholder="clinician@hospital.org" />
      <Field label="Phone" value={phone} onChangeText={setPhone} keyboardType="phone-pad" placeholder="+15551234567" />
      <Field label="Temporary Password" value={password} onChangeText={setPassword} secureTextEntry placeholder="At least 8 characters" />
      <Text style={styles.label}>Staff Role</Text>
      <Segmented
        value={role}
        options={[
          { value: "doctor", label: "Doctor" },
          { value: "nurse", label: "Nurse" },
          { value: "billing", label: "Billing" },
          { value: "admin", label: "Admin" },
        ]}
        onChange={setRole}
      />
      <PrimaryButton
        icon="person-add-outline"
        label={submitting ? "Registering..." : "Register Staff"}
        disabled={submitting}
        onPress={submit}
      />
    </View>
  );
}

const hospitalConsentPermissionOptions = [
  "HEALTH_RECORD_ACCESS",
  "AI_SUMMARIZATION",
  "SCHEDULING",
  "EMERGENCY_ESCALATION",
];

function HospitalConsentRequests({
  token,
  requests,
  loading,
  setRequests,
  onRefresh,
}: {
  token: string;
  requests: HospitalConsentRequest[];
  loading: boolean;
  setRequests: React.Dispatch<React.SetStateAction<HospitalConsentRequest[]>>;
  onRefresh: () => void;
}) {
  const [note, setNote] = useState("");
  const [expiresIn, setExpiresIn] = useState("30");
  const [permissions, setPermissions] = useState([
    "HEALTH_RECORD_ACCESS",
    "AI_SUMMARIZATION",
    "SCHEDULING",
  ]);
  const [qrDataURL, setQrDataURL] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  const togglePermission = (permissionName: string) => {
    setPermissions((current) =>
      current.includes(permissionName)
        ? current.filter((item) => item !== permissionName)
        : [...current, permissionName],
    );
  };

  const create = async () => {
    setSubmitting(true);
    setError("");
    try {
      const data = await apiRequest<{ request?: HospitalConsentRequest }>(
        endpoints.hospitalConsentRequests,
        {
          method: "POST",
          token,
          role: "hospital",
          body: {
            note: note.trim() || undefined,
            requested_permissions: permissions,
            expires_in_minutes: Number(expiresIn) || 30,
          },
        },
      );
      if (data.request) {
        setRequests((current) => [data.request as HospitalConsentRequest, ...current]);
        setQrDataURL(await QRCode.toDataURL(data.request.qr_payload || data.request.token || ""));
        setNote("");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Consent request failed.");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <View style={styles.stack}>
      <ScreenHeading
        title="Consent QR"
        subtitle="Generate a unique QR request that a patient can scan and approve."
      />
      <IconButton
        icon="refresh-outline"
        label={loading ? "Refreshing..." : "Refresh requests"}
        onPress={onRefresh}
        tone="neutral"
      />
      <Feedback error={error} />
      <Field label="Request Note" value={note} onChangeText={setNote} placeholder="Reason for access" />
      <Text style={styles.label}>Expires In</Text>
      <Segmented
        value={expiresIn}
        options={[
          { value: "15", label: "15m" },
          { value: "30", label: "30m" },
          { value: "60", label: "1h" },
          { value: "240", label: "4h" },
        ]}
        onChange={setExpiresIn}
      />
      <View style={styles.chipRow}>
        {hospitalConsentPermissionOptions.map((permissionName) => {
          const selected = permissions.includes(permissionName);
          return (
            <Pressable
              key={permissionName}
              onPress={() => togglePermission(permissionName)}
              style={[styles.chip, selected ? styles.chipActive : null]}
            >
              <Text style={[styles.chipText, selected ? styles.chipTextActive : null]}>
                {titleFromCode(permissionName)}
              </Text>
            </Pressable>
          );
        })}
      </View>
      <PrimaryButton
        icon="qr-code-outline"
        label={submitting ? "Generating..." : "Generate QR Request"}
        disabled={submitting || permissions.length === 0}
        onPress={create}
      />
      {qrDataURL ? (
        <View style={styles.qrCard}>
          <Image source={{ uri: qrDataURL }} style={styles.qrImage} />
          <Text style={styles.meta}>{requests[0]?.token}</Text>
        </View>
      ) : null}
      <Section title="Recent Requests">
        {requests.map((request) => (
          <View key={request.id || request.token} style={styles.card}>
            <View style={styles.rowBetween}>
              <View style={styles.flex}>
                <Text style={styles.cardTitle}>
                  {request.patient_id || "Patient claims on scan"}
                </Text>
                <Text style={styles.meta}>
                  {request.status || "pending"} • Expires {formatTime(request.expires_at)}
                </Text>
              </View>
              <Text style={styles.badge}>{request.status || "pending"}</Text>
            </View>
            <Text style={styles.cardBody}>{request.token}</Text>
          </View>
        ))}
        {requests.length === 0 ? (
          <EmptyText text="No consent requests generated yet." />
        ) : null}
      </Section>
    </View>
  );
}

function IncidentList({
  incidents,
  loading,
  onRefresh,
}: {
  incidents: Incident[];
  loading: boolean;
  onRefresh: () => void;
}) {
  return (
    <View style={styles.stack}>
      <ScreenHeading
        title="Emergency Safety Inbox"
        subtitle="Safety dispatch logs flagged during active AI voice triage sessions."
      />
      <IconButton
        icon="refresh-outline"
        label={loading ? "Refreshing..." : "Refresh alerts feed"}
        onPress={onRefresh}
        tone="neutral"
      />

      {incidents.map((incident) => {
        const critical =
          incident.severity?.toLowerCase() === "critical" ||
          incident.severity?.toLowerCase() === "high";
        return (
          <View
            key={
              incident.event_id ??
              `${incident.patient_id}-${incident.timestamp}`
            }
            style={[styles.card, critical ? styles.cardAlert : null]}
          >
            <View style={styles.rowBetween}>
              <Text
                style={[styles.cardTitle, critical ? styles.textAlert : null]}
              >
                {critical ? "⚠️ " : ""}
                {incident.severity || "Incident Alarm"}
              </Text>
              <Text style={[styles.badge, critical ? styles.badgeWarn : null]}>
                {incident.service_name || "voice service"}
              </Text>
            </View>
            <Text style={styles.cardBody}>
              Patient Ref: {incident.patient_id || "Unlisted"}
            </Text>
            <Text style={styles.meta}>
              Session: {incident.session_id || "None active"} |{" "}
              {formatTime(incident.timestamp)}
            </Text>
            {incident.failure_reason ? (
              <View style={styles.failureReasonBox}>
                <Text style={styles.errorText}>{incident.failure_reason}</Text>
              </View>
            ) : null}
          </View>
        );
      })}
      {incidents.length === 0 ? (
        <EmptyText
          text={loading ? "Loading incidents..." : "No incidents reported."}
        />
      ) : null}
    </View>
  );
}

function HospitalAudit({
  token,
  initialLookup = "",
  refreshSignal = 0,
  onRefreshComplete,
}: {
  token: string;
  initialLookup?: string;
  refreshSignal?: number;
  onRefreshComplete?: () => void;
}) {
  const [patientID, setPatientID] = useState(initialLookup);
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    setPatientID(initialLookup);
  }, [initialLookup]);

  const load = useCallback(async () => {
    if (!patientID.trim()) return;
    setLoading(true);
    setError("");
    try {
      const data = await apiRequest<{ events?: AuditEvent[] }>(
        `${endpoints.hospitalPatientAudit}?patient_id=${encodeURIComponent(patientID.trim())}`,
        { token },
      );
      setEvents(data.events ?? []);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Unable to load audit trail.",
      );
    } finally {
      setLoading(false);
    }
  }, [patientID, token]);

  useEffect(() => {
    if (!refreshSignal) return;
    void load().finally(onRefreshComplete);
  }, [load, onRefreshComplete, refreshSignal]);

  return (
    <View style={styles.stack}>
      <ScreenHeading
        title="Clinical Audit Lookup"
        subtitle="Audit compliance log tracking under clinical credentials."
      />
      <Field
        label="Search Patient"
        value={patientID}
        onChangeText={setPatientID}
        autoCapitalize="none"
        placeholder="Name, email, or patient ID"
      />
      <PrimaryButton
        icon="search-outline"
        label={loading ? "Tracing logs..." : "Load Audit Trail"}
        disabled={loading}
        onPress={load}
      />
      <Feedback error={error} />
      <AuditList events={events} compact />
    </View>
  );
}

function AuditList({
  events,
  compact = false,
}: {
  events: AuditEvent[];
  compact?: boolean;
}) {
  const [typeFilter, setTypeFilter] = useState("all");
  const [fromFilter, setFromFilter] = useState("");
  const [toFilter, setToFilter] = useState("");
  const [showTypeOptions, setShowTypeOptions] = useState(false);

  const eventTypes = useMemo(() => getAuditEventTypeOptions(events), [events]);
  const filteredEvents = useMemo(
    () => filterAuditEvents(events, typeFilter, fromFilter, toFilter),
    [events, fromFilter, toFilter, typeFilter],
  );
  const selectedTypeLabel =
    typeFilter === "all" ? "All audit types" : titleFromCode(typeFilter);

  return (
    <View style={styles.stack}>
      {!compact ? (
        <ScreenHeading
          title="Access Logs Trace"
          subtitle="Immutable activity ledger verification traces."
        />
      ) : null}
      <View style={styles.filterPanel}>
        <Text style={styles.label}>Event Type</Text>
        <Pressable
          style={styles.selectButton}
          onPress={() => setShowTypeOptions((current) => !current)}
        >
          <Text style={styles.selectButtonText}>{selectedTypeLabel}</Text>
          <Ionicons
            name={showTypeOptions ? "chevron-up" : "chevron-down"}
            size={16}
            color="#475569"
          />
        </Pressable>
        {showTypeOptions ? (
          <View style={styles.selectMenu}>
            {[
              { value: "all", label: "All audit types" },
              ...eventTypes.map((eventType) => ({
                value: eventType,
                label: titleFromCode(eventType),
              })),
            ].map((option) => (
              <Pressable
                key={option.value}
                style={[
                  styles.selectOption,
                  typeFilter === option.value
                    ? styles.selectOptionActive
                    : null,
                ]}
                onPress={() => {
                  setTypeFilter(option.value);
                  setShowTypeOptions(false);
                }}
              >
                <Text
                  style={[
                    styles.selectOptionText,
                    typeFilter === option.value
                      ? styles.selectOptionTextActive
                      : null,
                  ]}
                >
                  {option.label}
                </Text>
              </Pressable>
            ))}
          </View>
        ) : null}
        <View style={styles.gridTwo}>
          <View style={styles.flex}>
            <Field
              label="From"
              value={fromFilter}
              onChangeText={setFromFilter}
              placeholder="YYYY-MM-DD HH:mm"
            />
          </View>
          <View style={styles.flex}>
            <Field
              label="To"
              value={toFilter}
              onChangeText={setToFilter}
              placeholder="YYYY-MM-DD HH:mm"
            />
          </View>
        </View>
      </View>
      {filteredEvents.map((event, idx) => {
        const failed = event.success_status === false;
        return (
          <View
            key={event.event_id ?? `${event.event_type}-${idx}`}
            style={styles.card}
          >
            <View style={styles.rowBetween}>
              <Text style={styles.cardTitle}>
                {titleFromCode(event.event_type)}
              </Text>
              <Text style={[styles.badge, failed ? styles.badgeWarn : null]}>
                {failed ? "Access failed" : "Verified"}
              </Text>
            </View>
            <Text style={styles.cardBody}>
              Service: {event.service_name || "Record Manager"}
            </Text>
            <Text style={styles.meta}>{formatTime(event.timestamp)}</Text>
            {event.failure_reason ? (
              <Text style={styles.errorText}>
                Failure Details: {event.failure_reason}
              </Text>
            ) : null}
          </View>
        );
      })}
      {filteredEvents.length === 0 ? (
        <EmptyText text="No audit history matches the current filters." />
      ) : null}
    </View>
  );
}

function CallRow({ call }: { call: CallSummary }) {
  return (
    <View style={styles.card}>
      <View style={styles.rowBetween}>
        <View style={{ flexDirection: "row", alignItems: "center", gap: 6 }}>
          <Ionicons name="call-outline" size={16} color="#4f46e5" />
          <Text style={styles.cardTitle}>Call Ref #{call.id}</Text>
        </View>
        <Text style={styles.badge}>{call.status || "Completed"}</Text>
      </View>
      <Text style={styles.meta}>
        {formatTime(call.started_at)} •{" "}
        {call.ended_at ? formatTime(call.ended_at) : "Active triage"}
      </Text>
      <Text style={styles.cardBody}>
        {call.summary || "automated summary transcription file."}
      </Text>
      {call.recording_url ? (
        <Pressable
          onPress={() => Linking.openURL(call.recording_url as string)}
          style={styles.playRecordBtn}
        >
          <Ionicons name="play" size={12} color="#4f46e5" />
          <Text style={styles.playRecordBtnText}>Open Audio Recording</Text>
        </Pressable>
      ) : null}
    </View>
  );
}

function callNumber(phone?: string) {
  if (!phone) {
    Alert.alert(
      "Voice phone unavailable",
      "The backend did not return a voice support number.",
    );
    return;
  }
  Linking.openURL(`tel:${phone.replace(/[^\d+]/g, "")}`).catch(() => {
    Alert.alert("Unable to open phone dialer");
  });
}

function extractConsentToken(value: string) {
  const trimmed = value.trim();
  if (!trimmed) return "";
  try {
    const parsed = JSON.parse(trimmed) as { token?: string };
    if (parsed.token) return parsed.token.trim();
  } catch {
    // Raw token fallback.
  }
  return trimmed;
}

function parseLocationCommand(
  payload: unknown,
): { command: string; sessionID?: string } | null {
  try {
    const data = typeof payload === "string" ? JSON.parse(payload) : payload;
    if (!data || typeof data !== "object") return null;
    const record = data as Record<string, unknown>;
    const nested =
      record.data && typeof record.data === "object"
        ? (record.data as Record<string, unknown>)
        : {};
    const command = String(
      record.command ?? record.Command ?? record.type ?? "",
    );
    const sessionID = String(
      record.sessionID ??
        record.SessionID ??
        record.session_id ??
        nested.sessionID ??
        nested.SessionID ??
        nested.session_id ??
        "",
    );
    if (!command) return null;
    return { command, sessionID };
  } catch {
    return null;
  }
}
