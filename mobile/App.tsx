import { Ionicons } from "@expo/vector-icons";
import * as Location from "expo-location";
import * as SecureStore from "expo-secure-store";
import { StatusBar } from "expo-status-bar";
import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  Linking,
  Platform,
  Pressable,
  SafeAreaView,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View
} from "react-native";

const API_URL = process.env.EXPO_PUBLIC_API_URL ?? "http://localhost:8081";
const LOCATION_WS_URL = process.env.EXPO_PUBLIC_LOCATION_WS_URL ?? "ws://localhost:8090";

const endpoints = {
  patientLogin: "/api/v1/auth/patient/login",
  patientRegister: "/api/v1/auth/patient/register",
  patientVerifyEmail: "/api/v1/auth/patient/register/verify",
  patientVerifyOtp: "/api/v1/auth/patient/register/verify-otp",
  patientProfile: "/api/v1/patient/profile",
  patientConsents: "/api/v1/patient/consents",
  patientRecordsAnswer: "/api/v1/patient/records/answer",
  patientCalls: "/api/v1/patient/calls",
  patientWelfareChecks: "/api/v1/patient/welfare-checks",
  patientAudit: "/api/v1/patient/audit",
  hospitalLogin: "/api/v1/auth/hospital/login",
  hospitalSummary: "/api/v1/hospital/records/summary",
  hospitalIncidents: "/api/v1/hospital/incidents",
  hospitalPatientAudit: "/api/v1/hospital/patient/audit"
};

type Role = "patient" | "hospital";
type PatientTab = "home" | "consents" | "records" | "calls" | "welfare" | "audit" | "location";
type HospitalTab = "summary" | "incidents" | "audit";

type APIError = { code: string; message: string };
type APIResponse<T> = { data?: T; error?: APIError };

type PatientLoginData = { message?: string; access_token?: string; patient_id?: string };
type HospitalLoginData = { message?: string; access_token?: string; hospital_id?: string; role?: string };
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
type WelfareCheckReason =
  | "medication_reminder"
  | "mental_wellbeing"
  | "daily_checkup"
  | "symptom_follow_up"
  | "care_plan_reminder"
  | "other";
type WelfareCheck = {
  id?: string;
  patient_id?: string;
  scheduled_at?: string;
  timezone?: string;
  reason_code?: WelfareCheckReason | string;
  reason_detail?: string;
  status?: string;
  attempt_count?: number;
  latest_run_id?: string;
  latest_run_status?: string;
  latest_run_attempts?: number;
  latest_run_failure_reason?: string;
  livekit_room_id?: string;
  sip_participant_id?: string;
  created_at?: string;
  updated_at?: string;
  cancelled_at?: string;
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

const consentCopy: Record<string, { label: string; description: string }> = {
  VOICE_ASSISTANT_USE: {
    label: "Voice assistant",
    description: "Allows Zorba to support phone-based care conversations."
  },
  HEALTH_RECORD_ACCESS: {
    label: "Health records",
    description: "Allows answers and summaries to reference your records."
  },
  LOCATION_ACCESS: {
    label: "Emergency location",
    description: "Shares GPS only during active emergency voice sessions."
  },
  SMS_NOTIFICATION: {
    label: "SMS notifications",
    description: "Allows important care updates by text message."
  },
  EMAIL_NOTIFICATION: {
    label: "Email notifications",
    description: "Allows care updates and verification messages by email."
  },
  AI_SUMMARIZATION: {
    label: "AI summaries",
    description: "Allows Zorba to create concise clinical summaries."
  },
  THIRD_PARTY_MODEL_PROCESSING: {
    label: "Model processing",
    description: "Allows approved model providers to process limited context."
  }
};

const consentTypes = Object.keys(consentCopy);
const focusOptions = ["full", "medications", "allergies", "diagnoses"];
const welfareReasonCopy: Record<WelfareCheckReason, string> = {
  medication_reminder: "Medication",
  mental_wellbeing: "Mental wellbeing",
  daily_checkup: "Daily checkup",
  symptom_follow_up: "Symptom follow-up",
  care_plan_reminder: "Care plan",
  other: "Other"
};
const welfareReasons = Object.keys(welfareReasonCopy) as WelfareCheckReason[];

async function saveSecure(key: string, value: string) {
  await SecureStore.setItemAsync(key, value);
}

async function readSecure(key: string) {
  return SecureStore.getItemAsync(key);
}

async function deleteSecure(key: string) {
  await SecureStore.deleteItemAsync(key);
}

type MobileCacheEntry = {
  expiresAt: number;
  value?: unknown;
  promise?: Promise<unknown>;
};

const mobileResponseCache = new Map<string, MobileCacheEntry>();

function mobileCacheKey(endpoint: string, token?: string) {
  return `${token ? token.slice(-16) : "anon"}:${endpoint}`;
}

function clearAPIResponseCache() {
  mobileResponseCache.clear();
}

async function apiRequest<T>(
  endpoint: string,
  options: { method?: string; token?: string; body?: unknown; ttlMs?: number; force?: boolean } = {}
): Promise<T> {
  const method = options.method ?? "GET";
  const canCache = method === "GET" && options.body === undefined;
  const cacheKey = mobileCacheKey(endpoint, options.token);
  const now = Date.now();

  if (canCache && !options.force) {
    const cached = mobileResponseCache.get(cacheKey);
    if (cached?.value !== undefined && cached.expiresAt > now) {
      return cached.value as T;
    }
    if (cached?.promise) {
      return cached.promise as Promise<T>;
    }
  }

  const headers: Record<string, string> = {
    Accept: "application/json"
  };
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  if (options.token) {
    headers.Authorization = `Bearer ${options.token}`;
  }

  const request = fetch(`${API_URL}${endpoint}`, {
    method,
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body)
  }).then(async (response) => {
    const payload = (await response.json()) as APIResponse<T>;
    if (!response.ok) {
      throw new Error(payload.error?.message ?? "Request failed.");
    }
    const value = payload.data ?? ({} as T);
    if (canCache) {
      mobileResponseCache.set(cacheKey, {
        expiresAt: Date.now() + (options.ttlMs ?? 30_000),
        value
      });
    }
    return value;
  });

  if (canCache) {
    mobileResponseCache.set(cacheKey, {
      expiresAt: now + (options.ttlMs ?? 30_000),
      promise: request
    });
  }

  try {
    return await request;
  } catch (error) {
    if (canCache) mobileResponseCache.delete(cacheKey);
    throw error;
  }
}

function preloadAPI<T>(endpoint: string, token: string, ttlMs = 30_000) {
  void apiRequest<T>(endpoint, { token, ttlMs }).catch(() => {
    // Preloading is opportunistic.
  });
}

function formatTime(value?: string) {
  if (!value) return "Unknown time";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function titleFromCode(value?: string) {
  return value ? value.replaceAll("_", " ").toLowerCase() : "Unknown";
}

function defaultWelfareDateTime() {
  const date = new Date(Date.now() + 60 * 60 * 1000);
  date.setMinutes(Math.ceil(date.getMinutes() / 5) * 5, 0, 0);
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}

export default function App() {
  const [role, setRole] = useState<Role>("patient");
  const [patientToken, setPatientToken] = useState("");
  const [hospitalToken, setHospitalToken] = useState("");
  const [booting, setBooting] = useState(true);

  useEffect(() => {
    const load = async () => {
      const [storedPatientToken, storedHospitalToken] = await Promise.all([
        readSecure("patient_access_token"),
        readSecure("hospital_access_token")
      ]);
      setPatientToken(storedPatientToken ?? "");
      setHospitalToken(storedHospitalToken ?? "");
      if (storedHospitalToken && !storedPatientToken) {
        setRole("hospital");
      }
      setBooting(false);
    };
    void load();
  }, []);

  const signOut = async () => {
    await Promise.all([
      deleteSecure("patient_access_token"),
      deleteSecure("patient_id"),
      deleteSecure("hospital_access_token")
    ]);
    clearAPIResponseCache();
    setPatientToken("");
    setHospitalToken("");
  };

  if (booting) {
    return (
      <SafeAreaView style={styles.safeArea}>
        <View style={styles.center}>
          <ActivityIndicator />
          <Text style={styles.muted}>Loading secure session...</Text>
        </View>
      </SafeAreaView>
    );
  }

  return (
    <SafeAreaView style={styles.safeArea}>
      <StatusBar style="dark" />
      <View style={styles.shell}>
        <Header role={role} onRoleChange={setRole} signedIn={Boolean(patientToken || hospitalToken)} onSignOut={signOut} />
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
      </View>
    </SafeAreaView>
  );
}

function Header({
  role,
  onRoleChange,
  signedIn,
  onSignOut
}: {
  role: Role;
  onRoleChange: (role: Role) => void;
  signedIn: boolean;
  onSignOut: () => void;
}) {
  return (
    <View style={styles.header}>
      <View style={styles.headerBrand}>
        <View style={styles.brandMark}>
          <Ionicons name="pulse-outline" size={21} color="#ffffff" />
        </View>
        <View>
          <Text style={styles.brand}>Zorba Health</Text>
          <Text style={styles.headerSub}>{role === "patient" ? "Patient mobile care" : "Hospital staff console"}</Text>
        </View>
      </View>
      <View style={styles.headerActions}>
        {!signedIn ? (
          <Segmented
            value={role}
            options={[
              { value: "patient", label: "Patient" },
              { value: "hospital", label: "Hospital" }
            ]}
            onChange={(value) => onRoleChange(value as Role)}
          />
        ) : (
          <IconButton icon="log-out-outline" label="Sign out" onPress={onSignOut} tone="neutral" />
        )}
      </View>
    </View>
  );
}

function PatientAuth({ onLogin }: { onLogin: (token: string) => void }) {
  const [mode, setMode] = useState<"login" | "register" | "otp" | "email">("login");
  const [phone, setPhone] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [fullName, setFullName] = useState("");
  const [dateOfBirth, setDateOfBirth] = useState("");
  const [otp, setOtp] = useState("");
  const [emailToken, setEmailToken] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const login = async () => {
    setLoading(true);
    setError("");
    try {
      const data = await apiRequest<PatientLoginData>(endpoints.patientLogin, {
        method: "POST",
        body: { phone_number: phone.trim(), email: email.trim(), password }
      });
      if (!data.access_token) throw new Error("Login succeeded but no patient token was returned.");
      await saveSecure("patient_access_token", data.access_token);
      if (data.patient_id) await saveSecure("patient_id", data.patient_id);
      onLogin(data.access_token);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed.");
    } finally {
      setLoading(false);
    }
  };

  const register = async () => {
    setLoading(true);
    setError("");
    setNotice("");
    try {
      await apiRequest(endpoints.patientRegister, {
        method: "POST",
        body: {
          phone_number: phone.trim(),
          email: email.trim(),
          password,
          full_name: fullName.trim(),
          date_of_birth: dateOfBirth ? new Date(dateOfBirth).toISOString() : undefined
        }
      });
      setNotice("Registration started. Verify the phone OTP, then paste the email verification token if needed.");
      setMode("otp");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Registration failed.");
    } finally {
      setLoading(false);
    }
  };

  const verifyOtp = async () => {
    setLoading(true);
    setError("");
    try {
      await apiRequest(endpoints.patientVerifyOtp, {
        method: "POST",
        body: { phone_number: phone.trim(), otp: otp.trim() }
      });
      setNotice("Phone verified. Continue to email verification or sign in if your account is ready.");
      setMode("email");
    } catch (err) {
      setError(err instanceof Error ? err.message : "OTP verification failed.");
    } finally {
      setLoading(false);
    }
  };

  const verifyEmail = async () => {
    setLoading(true);
    setError("");
    try {
      await apiRequest(endpoints.patientVerifyEmail, {
        method: "POST",
        body: { token: emailToken.trim() }
      });
      setNotice("Email verified. You can sign in now.");
      setMode("login");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Email verification failed.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <ScrollView contentContainerStyle={styles.authScroll}>
      <View style={styles.authPanel}>
        <Text style={styles.screenTitle}>
          {mode === "login" ? "Patient sign in" : mode === "register" ? "Create patient account" : mode === "otp" ? "Verify phone" : "Verify email"}
        </Text>
        <Text style={styles.screenCopy}>Secure mobile access to Zorba voice support, consent controls, call summaries, and health-record answers.</Text>
        <View style={styles.stack}>
          {mode !== "email" ? <Field label="Phone number" value={phone} onChangeText={setPhone} keyboardType="phone-pad" placeholder="+1 555 123 4567" /> : null}
          {(mode === "login" || mode === "register") && (
            <>
              <Field label="Email" value={email} onChangeText={setEmail} keyboardType="email-address" autoCapitalize="none" />
              <Field label="Password" value={password} onChangeText={setPassword} secureTextEntry />
            </>
          )}
          {mode === "register" && (
            <>
              <Field label="Full name" value={fullName} onChangeText={setFullName} />
              <Field label="Date of birth" value={dateOfBirth} onChangeText={setDateOfBirth} placeholder="YYYY-MM-DD" />
            </>
          )}
          {mode === "otp" ? <Field label="One-time code" value={otp} onChangeText={setOtp} keyboardType="number-pad" /> : null}
          {mode === "email" ? <Field label="Email verification token" value={emailToken} onChangeText={setEmailToken} autoCapitalize="none" /> : null}
        </View>
        <PrimaryButton
          icon={mode === "login" ? "log-in-outline" : "checkmark-circle-outline"}
          label={loading ? "Working..." : mode === "login" ? "Sign in" : mode === "register" ? "Start registration" : mode === "otp" ? "Verify OTP" : "Verify email"}
          disabled={loading}
          onPress={mode === "login" ? login : mode === "register" ? register : mode === "otp" ? verifyOtp : verifyEmail}
        />
        <View style={styles.inlineActions}>
          <TextButton label={mode === "login" ? "Create account" : "Back to sign in"} onPress={() => setMode(mode === "login" ? "register" : "login")} />
          <TextButton label="Enter OTP" onPress={() => setMode("otp")} />
          <TextButton label="Email token" onPress={() => setMode("email")} />
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
      const data = await apiRequest<HospitalLoginData>(endpoints.hospitalLogin, {
        method: "POST",
        body: { email: email.trim(), password }
      });
      if (!data.access_token) throw new Error("Login succeeded but no hospital token was returned.");
      await saveSecure("hospital_access_token", data.access_token);
      onLogin(data.access_token);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Hospital login failed.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <ScrollView contentContainerStyle={styles.authScroll}>
      <View style={styles.authPanel}>
        <Text style={styles.screenTitle}>Hospital staff sign in</Text>
        <Text style={styles.screenCopy}>Review emergency incidents, generate patient summaries, and inspect patient audit activity.</Text>
        <View style={styles.stack}>
          <Field label="Email" value={email} onChangeText={setEmail} keyboardType="email-address" autoCapitalize="none" />
          <Field label="Password" value={password} onChangeText={setPassword} secureTextEntry />
        </View>
        <PrimaryButton icon="shield-checkmark-outline" label={loading ? "Signing in..." : "Sign in"} disabled={loading} onPress={login} />
        <Feedback error={error} />
      </View>
    </ScrollView>
  );
}

function PatientPortal({ token, onSignOut }: { token: string; onSignOut: () => void }) {
  const [tab, setTab] = useState<PatientTab>("home");
  const [profile, setProfile] = useState<PatientProfile | null>(null);
  const [consents, setConsents] = useState<ConsentRecord[]>([]);
  const [calls, setCalls] = useState<CallSummary[]>([]);
  const [welfareChecks, setWelfareChecks] = useState<WelfareCheck[]>([]);
  const [audit, setAudit] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const [profileData, consentData, callData, welfareData, auditData] = await Promise.all([
        apiRequest<PatientProfile>(endpoints.patientProfile, { token }),
        apiRequest<{ consents?: ConsentRecord[] }>(endpoints.patientConsents, { token }),
        apiRequest<{ calls?: CallSummary[] }>(endpoints.patientCalls, { token }),
        apiRequest<{ welfare_checks?: WelfareCheck[] }>(endpoints.patientWelfareChecks, { token }),
        apiRequest<{ events?: AuditEvent[] }>(endpoints.patientAudit, { token })
      ]);
      setProfile(profileData);
      setConsents(consentData.consents ?? []);
      setCalls(callData.calls ?? []);
      setWelfareChecks(welfareData.welfare_checks ?? []);
      setAudit(auditData.events ?? []);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unable to load patient data.";
      setError(message);
      if (message.toLowerCase().includes("token")) onSignOut();
    } finally {
      setLoading(false);
    }
  }, [onSignOut, token]);

  useEffect(() => {
    preloadAPI<PatientProfile>(endpoints.patientProfile, token);
    preloadAPI<{ consents?: ConsentRecord[] }>(endpoints.patientConsents, token);
    preloadAPI<{ calls?: CallSummary[] }>(endpoints.patientCalls, token);
    preloadAPI<{ welfare_checks?: WelfareCheck[] }>(endpoints.patientWelfareChecks, token);
    preloadAPI<{ events?: AuditEvent[] }>(endpoints.patientAudit, token);
    void load();
  }, [load, token]);

  const activeConsents = useMemo(() => {
    const map = new Map<string, ConsentRecord>();
    for (const consent of consents) {
      if (consent.consent_type && !consent.revoked_at && consent.status !== "revoked") {
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
          { value: "records", label: "Ask", icon: "chatbubbles-outline" },
          { value: "calls", label: "Calls", icon: "call-outline" },
          { value: "welfare", label: "Checks", icon: "calendar-outline" },
          { value: "audit", label: "Audit", icon: "reader-outline" },
          { value: "location", label: "GPS", icon: "navigate-outline" }
        ]}
      />
      <ScrollView contentContainerStyle={styles.content}>
        {loading ? <LoadingCard /> : null}
        <Feedback error={error} />
        {!loading && tab === "home" ? <PatientHome profile={profile} calls={calls} audit={audit} onRefresh={load} /> : null}
        {!loading && tab === "consents" ? <ConsentCenter token={token} consents={consents} setConsents={setConsents} /> : null}
        {!loading && tab === "records" ? <HealthQuestion token={token} /> : null}
        {!loading && tab === "calls" ? <CallList calls={calls} voicePhone={profile?.voice_phone} /> : null}
        {!loading && tab === "welfare" ? <WelfareChecks token={token} checks={welfareChecks} setChecks={setWelfareChecks} /> : null}
        {!loading && tab === "audit" ? <AuditList events={audit} /> : null}
        {!loading && tab === "location" ? <LocationSharing token={token} enabled={activeConsents.has("LOCATION_ACCESS")} /> : null}
      </ScrollView>
    </View>
  );
}

function PatientHome({
  profile,
  calls,
  audit,
  onRefresh
}: {
  profile: PatientProfile | null;
  calls: CallSummary[];
  audit: AuditEvent[];
  onRefresh: () => void;
}) {
  return (
    <View style={styles.stack}>
      <View style={styles.heroPanel}>
        <Text style={styles.eyebrow}>Patient home</Text>
        <Text style={styles.largeTitle}>{profile?.full_name || "Welcome"}</Text>
        <Text style={styles.heroCopy}>{profile?.support_window || "24/7"} support through {profile?.voice_phone || "the Zorba voice line"}.</Text>
        <View style={styles.heroActions}>
          <PrimaryButton icon="call-outline" label="Call Zorba" onPress={() => callNumber(profile?.voice_phone)} />
          <IconButton icon="refresh-outline" label="Refresh" onPress={onRefresh} tone="neutral" />
        </View>
      </View>
      <View style={styles.gridTwo}>
        <InfoCard icon="person-outline" title="Profile" body={`${profile?.phone_number || "No phone"}\n${profile?.email || "No email"}`} />
        <InfoCard icon="document-text-outline" title="Recent activity" body={audit[0] ? `${titleFromCode(audit[0].event_type)}\n${formatTime(audit[0].timestamp)}` : "No audit events yet."} />
      </View>
      <Section title="Recent calls">
        {calls.slice(0, 3).map((call) => (
          <CallRow key={String(call.id)} call={call} />
        ))}
        {calls.length === 0 ? <EmptyText text="No call summaries yet." /> : null}
      </Section>
    </View>
  );
}

function ConsentCenter({
  token,
  consents,
  setConsents
}: {
  token: string;
  consents: ConsentRecord[];
  setConsents: (items: ConsentRecord[]) => void;
}) {
  const [mutating, setMutating] = useState("");
  const [error, setError] = useState("");

  const active = useMemo(() => {
    const map = new Map<string, ConsentRecord>();
    for (const consent of consents) {
      if (consent.consent_type && !consent.revoked_at && consent.status !== "revoked") {
        map.set(consent.consent_type, consent);
      }
    }
    return map;
  }, [consents]);

  const mutate = async (type: string, grant: boolean) => {
    setMutating(type);
    setError("");
    try {
      const data = await apiRequest<{ consent?: ConsentRecord }>(endpoints.patientConsents, {
        method: grant ? "POST" : "DELETE",
        token,
        body: { consent_type: type, source: "patient-mobile-app" }
      });
      if (data.consent) {
        setConsents([data.consent, ...consents.filter((item) => item.consent_type !== type)]);
      }
      clearAPIResponseCache();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Consent update failed.");
    } finally {
      setMutating("");
    }
  };

  return (
    <View style={styles.stack}>
      <ScreenHeading title="Consent center" subtitle="Review every consent type modeled by the backend and change it directly from mobile." />
      <Feedback error={error} />
      {consentTypes.map((type) => {
        const granted = active.has(type);
        return (
          <View key={type} style={styles.card}>
            <View style={styles.rowBetween}>
              <View style={styles.flex}>
                <Text style={styles.cardTitle}>{consentCopy[type].label}</Text>
                <Text style={styles.cardBody}>{consentCopy[type].description}</Text>
              </View>
              <Pressable
                accessibilityRole="switch"
                accessibilityState={{ checked: granted }}
                disabled={mutating === type}
                onPress={() => mutate(type, !granted)}
                style={[styles.switchTrack, granted ? styles.switchOn : styles.switchOff]}
              >
                <View style={[styles.switchThumb, granted ? styles.switchThumbOn : styles.switchThumbOff]} />
              </Pressable>
            </View>
            <Text style={styles.meta}>{granted ? `Granted ${formatTime(active.get(type)?.granted_at)}` : "Not currently granted"}</Text>
          </View>
        );
      })}
    </View>
  );
}

function HealthQuestion({ token }: { token: string }) {
  const [question, setQuestion] = useState("");
  const [answer, setAnswer] = useState("");
  const [citations, setCitations] = useState<Citation[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const ask = async () => {
    if (!question.trim()) return;
    setLoading(true);
    setError("");
    setAnswer("");
    setCitations([]);
    try {
      const data = await apiRequest<{ answer?: string; citations?: Citation[] }>(endpoints.patientRecordsAnswer, {
        method: "POST",
        token,
        body: { question: question.trim(), top_k: 5 }
      });
      setAnswer(data.answer ?? "No answer returned.");
      setCitations(data.citations ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to answer that question.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <View style={styles.stack}>
      <ScreenHeading title="Ask your records" subtitle="Questions are answered from the backend health-record retrieval flow." />
      <Field label="Question" value={question} onChangeText={setQuestion} multiline placeholder="What medications are in my record?" />
      <PrimaryButton icon="send-outline" label={loading ? "Asking..." : "Ask Zorba"} disabled={loading} onPress={ask} />
      <Feedback error={error} />
      {answer ? (
        <Section title="Answer">
          <Text style={styles.bodyText}>{answer}</Text>
        </Section>
      ) : null}
      {citations.length > 0 ? (
        <Section title="Sources">
          {citations.map((citation, index) => (
            <View key={`${citation.source_file}-${index}`} style={styles.subCard}>
              <Text style={styles.cardTitle}>{citation.source_file || "Record source"}</Text>
              <Text style={styles.cardBody}>{citation.text || "No snippet returned."}</Text>
            </View>
          ))}
        </Section>
      ) : null}
    </View>
  );
}

function CallList({ calls, voicePhone }: { calls: CallSummary[]; voicePhone?: string }) {
  return (
    <View style={styles.stack}>
      <ScreenHeading title="Call summaries" subtitle="Phone-based voice support remains the entry point while in-app calling is deferred." />
      <PrimaryButton icon="call-outline" label="Call Zorba" onPress={() => callNumber(voicePhone)} />
      <Section title="Recent calls">
        {calls.map((call) => (
          <CallRow key={String(call.id)} call={call} />
        ))}
        {calls.length === 0 ? <EmptyText text="No call summaries have been returned by the backend." /> : null}
      </Section>
    </View>
  );
}

function WelfareChecks({
  token,
  checks,
  setChecks
}: {
  token: string;
  checks: WelfareCheck[];
  setChecks: (items: WelfareCheck[]) => void;
}) {
  const [scheduledAt, setScheduledAt] = useState(defaultWelfareDateTime);
  const [reason, setReason] = useState<WelfareCheckReason>("daily_checkup");
  const [detail, setDetail] = useState("");
  const [loading, setLoading] = useState(false);
  const [cancelling, setCancelling] = useState("");
  const [error, setError] = useState("");

  const create = async () => {
    const scheduled = new Date(scheduledAt);
    if (Number.isNaN(scheduled.getTime())) {
      setError("Enter a valid date and time.");
      return;
    }
    if (detail.length > 1000) {
      setError("Detail must be 1000 characters or less.");
      return;
    }
    setLoading(true);
    setError("");
    try {
      const data = await apiRequest<{ welfare_check?: WelfareCheck }>(endpoints.patientWelfareChecks, {
        method: "POST",
        token,
        body: {
          scheduled_at: scheduled.toISOString(),
          timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
          reason_code: reason,
          reason_detail: detail.trim()
        }
      });
      if (data.welfare_check) {
        setChecks([data.welfare_check, ...checks]);
      }
      clearAPIResponseCache();
      setDetail("");
      setScheduledAt(defaultWelfareDateTime());
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to schedule welfare check.");
    } finally {
      setLoading(false);
    }
  };

  const cancel = async (id?: string) => {
    if (!id) return;
    setCancelling(id);
    setError("");
    try {
      const data = await apiRequest<{ welfare_check?: WelfareCheck }>(`${endpoints.patientWelfareChecks}/${encodeURIComponent(id)}`, {
        method: "DELETE",
        token
      });
      if (data.welfare_check) {
        setChecks(checks.map((item) => (item.id === id ? data.welfare_check! : item)));
      }
      clearAPIResponseCache();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to cancel welfare check.");
    } finally {
      setCancelling("");
    }
  };

  return (
    <View style={styles.stack}>
      <ScreenHeading title="Welfare checks" subtitle="Schedule outbound phone check-ins using your saved patient number." />
      <Feedback error={error} />
      <Section title="New check">
        <Field label="Date and time" value={scheduledAt} onChangeText={setScheduledAt} placeholder="YYYY-MM-DDTHH:mm" />
        <Segmented
          value={reason}
          onChange={(value) => setReason(value as WelfareCheckReason)}
          options={welfareReasons.map((item) => ({ value: item, label: welfareReasonCopy[item] }))}
        />
        <Field label="Detail" value={detail} onChangeText={setDetail} multiline placeholder="Anything Zorba should know before calling?" />
        <Text style={styles.meta}>{detail.length}/1000</Text>
        <PrimaryButton icon="calendar-outline" label={loading ? "Scheduling..." : "Schedule check"} disabled={loading} onPress={create} />
      </Section>
      <Section title="Scheduled and recent">
        {checks.map((check) => {
          const cancellable = ["scheduled", "pending"].includes((check.status || "").toLowerCase());
          return (
            <View key={check.id ?? `${check.scheduled_at}-${check.reason_code}`} style={styles.card}>
              <View style={styles.rowBetween}>
                <View style={styles.flex}>
                  <Text style={styles.cardTitle}>{welfareReasonCopy[check.reason_code as WelfareCheckReason] ?? titleFromCode(check.reason_code)}</Text>
                  <Text style={styles.meta}>{formatTime(check.scheduled_at)}</Text>
                </View>
                <Text style={styles.badge}>{check.latest_run_status || check.status || "scheduled"}</Text>
              </View>
              {check.reason_detail ? <Text style={styles.cardBody}>{check.reason_detail}</Text> : null}
              {check.latest_run_failure_reason ? <Text style={styles.errorText}>{check.latest_run_failure_reason}</Text> : null}
              {typeof check.latest_run_attempts === "number" && check.latest_run_attempts > 0 ? (
                <Text style={styles.meta}>Attempts: {check.latest_run_attempts}</Text>
              ) : null}
              {cancellable ? (
                <IconButton
                  icon="close-circle-outline"
                  label={cancelling === check.id ? "Cancelling" : "Cancel"}
                  onPress={() => cancel(check.id)}
                  tone="neutral"
                />
              ) : null}
            </View>
          );
        })}
        {checks.length === 0 ? <EmptyText text="No welfare checks scheduled yet." /> : null}
      </Section>
    </View>
  );
}

function LocationSharing({ token, enabled }: { token: string; enabled: boolean }) {
  const [status, setStatus] = useState(enabled ? "Waiting for a voice session." : "Enable emergency location consent first.");
  const [connected, setConnected] = useState(false);
  const [activeSession, setActiveSession] = useState("");
  const wsRef = useRef<WebSocket | null>(null);
  const subscriptionRef = useRef<Location.LocationSubscription | null>(null);

  const stopWatch = useCallback(() => {
    subscriptionRef.current?.remove();
    subscriptionRef.current = null;
    setActiveSession("");
  }, []);

  useEffect(() => {
    if (!enabled) {
      stopWatch();
      wsRef.current?.close();
      setConnected(false);
      setStatus("Enable emergency location consent first.");
      return;
    }

    const wsURL = `${LOCATION_WS_URL.replace(/\/$/, "")}/ws/location?token=${encodeURIComponent(token)}`;
    const ws = new WebSocket(wsURL);
    wsRef.current = ws;
    ws.onopen = () => {
      setConnected(true);
      setStatus("Connected. GPS starts only when a voice session asks for it.");
    };
    ws.onclose = () => {
      setConnected(false);
      stopWatch();
      setStatus("Location channel closed.");
    };
    ws.onerror = () => {
      setStatus("Location channel error.");
    };
    ws.onmessage = async (event) => {
      const command = parseLocationCommand(event.data);
      if (!command) return;
      if (command.command === "stop_location") {
        stopWatch();
        setStatus("Location sharing stopped.");
        return;
      }
      if (command.command === "start_location" && command.sessionID) {
        setActiveSession(command.sessionID);
        const permission = await Location.requestForegroundPermissionsAsync();
        if (permission.status !== "granted") {
          setStatus("Location permission was not granted.");
          return;
        }
        stopWatch();
        subscriptionRef.current = await Location.watchPositionAsync(
          {
            accuracy: Location.Accuracy.High,
            timeInterval: 5000,
            distanceInterval: 10
          },
          (position) => {
            if (ws.readyState === WebSocket.OPEN) {
              ws.send(
                JSON.stringify({
                  type: "location_update",
                  sessionID: command.sessionID,
                  lat: position.coords.latitude,
                  lng: position.coords.longitude,
                  accuracy: position.coords.accuracy ?? 0,
                  method: "gps"
                })
              );
            }
          }
        );
        setStatus("Sharing live GPS for the active voice session.");
      }
    };

    return () => {
      stopWatch();
      ws.close();
    };
  }, [enabled, stopWatch, token]);

  return (
    <View style={styles.stack}>
      <ScreenHeading title="Emergency location" subtitle="The websocket is kept ready, but coordinates are sent only after the backend starts an emergency voice session." />
      <View style={styles.card}>
        <View style={styles.rowBetween}>
          <View>
            <Text style={styles.cardTitle}>{connected ? "Connected" : "Disconnected"}</Text>
            <Text style={styles.cardBody}>{status}</Text>
          </View>
          <Ionicons name={connected ? "radio-outline" : "radio-button-off-outline"} size={28} color={connected ? "#047857" : "#64748b"} />
        </View>
        <Text style={styles.meta}>Active session: {activeSession || "none"}</Text>
      </View>
    </View>
  );
}

function HospitalPortal({ token, onSignOut }: { token: string; onSignOut: () => void }) {
  const [tab, setTab] = useState<HospitalTab>("summary");
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [loadingIncidents, setLoadingIncidents] = useState(false);

  const loadIncidents = useCallback(async () => {
    setLoadingIncidents(true);
    try {
      const data = await apiRequest<{ incidents?: Incident[] }>(endpoints.hospitalIncidents, { token });
      setIncidents(data.incidents ?? []);
    } catch (err) {
      if (err instanceof Error && err.message.toLowerCase().includes("token")) onSignOut();
    } finally {
      setLoadingIncidents(false);
    }
  }, [onSignOut, token]);

  useEffect(() => {
    preloadAPI<{ incidents?: Incident[] }>(endpoints.hospitalIncidents, token);
    void loadIncidents();
  }, [loadIncidents, token]);

  return (
    <View style={styles.portal}>
      <TabBar
        value={tab}
        onChange={(value) => setTab(value as HospitalTab)}
        options={[
          { value: "summary", label: "Summary", icon: "medkit-outline" },
          { value: "incidents", label: "Incidents", icon: "alert-circle-outline" },
          { value: "audit", label: "Audit", icon: "reader-outline" }
        ]}
      />
      <ScrollView contentContainerStyle={styles.content}>
        {tab === "summary" ? <HospitalSummary token={token} /> : null}
        {tab === "incidents" ? <IncidentList incidents={incidents} loading={loadingIncidents} onRefresh={loadIncidents} /> : null}
        {tab === "audit" ? <HospitalAudit token={token} /> : null}
      </ScrollView>
    </View>
  );
}

function HospitalSummary({ token }: { token: string }) {
  const [patientID, setPatientID] = useState("");
  const [focus, setFocus] = useState("full");
  const [summary, setSummary] = useState("");
  const [audit, setAudit] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const generate = async () => {
    if (!patientID.trim()) return;
    setLoading(true);
    setError("");
    setSummary("");
    setAudit([]);
    try {
      const data = await apiRequest<{ summary?: string }>(endpoints.hospitalSummary, {
        method: "POST",
        token,
        body: { patient_id: patientID.trim(), focus }
      });
      setSummary(data.summary ?? "No summary returned.");
      const auditData = await apiRequest<{ events?: AuditEvent[] }>(
        `${endpoints.hospitalPatientAudit}?patient_id=${encodeURIComponent(patientID.trim())}`,
        { token, force: true }
      );
      setAudit(auditData.events ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to summarize patient records.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <View style={styles.stack}>
      <ScreenHeading title="Patient record summary" subtitle="Generate staff-facing summaries using the backend record summarizer." />
      <Field label="Patient ID" value={patientID} onChangeText={setPatientID} autoCapitalize="none" />
      <Segmented value={focus} options={focusOptions.map((value) => ({ value, label: value }))} onChange={setFocus} />
      <PrimaryButton icon="sparkles-outline" label={loading ? "Generating..." : "Generate summary"} disabled={loading} onPress={generate} />
      <Feedback error={error} />
      {summary ? (
        <Section title="Summary">
          <Text style={styles.bodyText}>{summary}</Text>
        </Section>
      ) : null}
      {audit.length > 0 ? <AuditList events={audit} compact /> : null}
    </View>
  );
}

function IncidentList({
  incidents,
  loading,
  onRefresh
}: {
  incidents: Incident[];
  loading: boolean;
  onRefresh: () => void;
}) {
  return (
    <View style={styles.stack}>
      <ScreenHeading title="Emergency incidents" subtitle="Recent emergency events visible to the logged-in hospital." />
      <IconButton icon="refresh-outline" label={loading ? "Refreshing" : "Refresh"} onPress={onRefresh} tone="neutral" />
      {incidents.map((incident) => (
        <View key={incident.event_id ?? `${incident.patient_id}-${incident.timestamp}`} style={styles.card}>
          <View style={styles.rowBetween}>
            <Text style={styles.cardTitle}>{incident.severity || "Incident"}</Text>
            <Text style={styles.badge}>{incident.service_name || "service"}</Text>
          </View>
          <Text style={styles.cardBody}>Patient: {incident.patient_id || "Unknown"}</Text>
          <Text style={styles.meta}>Session: {incident.session_id || "none"} | {formatTime(incident.timestamp)}</Text>
          {incident.failure_reason ? <Text style={styles.errorText}>{incident.failure_reason}</Text> : null}
        </View>
      ))}
      {incidents.length === 0 ? <EmptyText text={loading ? "Loading incidents..." : "No incidents returned."} /> : null}
    </View>
  );
}

function HospitalAudit({ token }: { token: string }) {
  const [patientID, setPatientID] = useState("");
  const [events, setEvents] = useState<AuditEvent[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const load = async () => {
    if (!patientID.trim()) return;
    setLoading(true);
    setError("");
    try {
      const data = await apiRequest<{ events?: AuditEvent[] }>(
        `${endpoints.hospitalPatientAudit}?patient_id=${encodeURIComponent(patientID.trim())}`,
        { token }
      );
      setEvents(data.events ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to load audit trail.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <View style={styles.stack}>
      <ScreenHeading title="Patient audit trail" subtitle="Look up patient-specific audit events allowed by the hospital token." />
      <Field label="Patient ID" value={patientID} onChangeText={setPatientID} autoCapitalize="none" />
      <PrimaryButton icon="search-outline" label={loading ? "Loading..." : "Load audit"} disabled={loading} onPress={load} />
      <Feedback error={error} />
      <AuditList events={events} compact />
    </View>
  );
}

function AuditList({ events, compact = false }: { events: AuditEvent[]; compact?: boolean }) {
  return (
    <View style={styles.stack}>
      {!compact ? <ScreenHeading title="Audit trail" subtitle="Recent backend audit events for this account." /> : null}
      {events.map((event) => (
        <View key={event.event_id ?? `${event.event_type}-${event.timestamp}`} style={styles.card}>
          <View style={styles.rowBetween}>
            <Text style={styles.cardTitle}>{titleFromCode(event.event_type)}</Text>
            <Text style={[styles.badge, event.success_status === false ? styles.badgeWarn : null]}>
              {event.success_status === false ? "failed" : "ok"}
            </Text>
          </View>
          <Text style={styles.cardBody}>Service: {event.service_name || "Unknown"}</Text>
          <Text style={styles.meta}>{formatTime(event.timestamp)}</Text>
          {event.failure_reason ? <Text style={styles.errorText}>{event.failure_reason}</Text> : null}
        </View>
      ))}
      {events.length === 0 ? <EmptyText text="No audit events returned." /> : null}
    </View>
  );
}

function CallRow({ call }: { call: CallSummary }) {
  return (
    <View style={styles.card}>
      <View style={styles.rowBetween}>
        <Text style={styles.cardTitle}>Call #{call.id}</Text>
        <Text style={styles.badge}>{call.status || "unknown"}</Text>
      </View>
      <Text style={styles.meta}>{formatTime(call.started_at)} - {call.ended_at ? formatTime(call.ended_at) : "active or unknown"}</Text>
      <Text style={styles.cardBody}>{call.summary || "No summary returned for this call."}</Text>
      {call.recording_url ? <TextButton label="Open recording" onPress={() => Linking.openURL(call.recording_url as string)} /> : null}
    </View>
  );
}

function Field(props: {
  label: string;
  value: string;
  onChangeText: (value: string) => void;
  placeholder?: string;
  secureTextEntry?: boolean;
  keyboardType?: "default" | "email-address" | "number-pad" | "phone-pad";
  autoCapitalize?: "none" | "sentences" | "words" | "characters";
  multiline?: boolean;
}) {
  return (
    <View style={styles.field}>
      <Text style={styles.label}>{props.label}</Text>
      <TextInput
        style={[styles.input, props.multiline ? styles.textArea : null]}
        value={props.value}
        onChangeText={props.onChangeText}
        placeholder={props.placeholder}
        secureTextEntry={props.secureTextEntry}
        keyboardType={props.keyboardType}
        autoCapitalize={props.autoCapitalize}
        multiline={props.multiline}
        placeholderTextColor="#94a3b8"
      />
    </View>
  );
}

function PrimaryButton({ icon, label, onPress, disabled = false }: { icon: keyof typeof Ionicons.glyphMap; label: string; onPress: () => void; disabled?: boolean }) {
  return (
    <Pressable disabled={disabled} onPress={onPress} style={[styles.primaryButton, disabled ? styles.disabled : null]}>
      <Ionicons name={icon} size={18} color="#ffffff" />
      <Text style={styles.primaryText}>{label}</Text>
    </Pressable>
  );
}

function IconButton({
  icon,
  label,
  onPress,
  tone
}: {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  tone: "neutral" | "accent";
}) {
  return (
    <Pressable onPress={onPress} style={[styles.iconButton, tone === "accent" ? styles.iconAccent : null]}>
      <Ionicons name={icon} size={17} color={tone === "accent" ? "#ffffff" : "#334155"} />
      <Text style={[styles.iconButtonText, tone === "accent" ? styles.iconAccentText : null]}>{label}</Text>
    </Pressable>
  );
}

function TextButton({ label, onPress }: { label: string; onPress: () => void }) {
  return (
    <Pressable onPress={onPress} style={styles.textButton}>
      <Text style={styles.textButtonLabel}>{label}</Text>
    </Pressable>
  );
}

function Segmented({ value, options, onChange }: { value: string; options: { value: string; label: string }[]; onChange: (value: string) => void }) {
  return (
    <View style={styles.segmented}>
      {options.map((option) => {
        const active = value === option.value;
        return (
          <Pressable key={option.value} onPress={() => onChange(option.value)} style={[styles.segment, active ? styles.segmentActive : null]}>
            <Text style={[styles.segmentText, active ? styles.segmentTextActive : null]}>{option.label}</Text>
          </Pressable>
        );
      })}
    </View>
  );
}

function TabBar({
  value,
  options,
  onChange
}: {
  value: string;
  options: { value: string; label: string; icon: keyof typeof Ionicons.glyphMap }[];
  onChange: (value: string) => void;
}) {
  return (
    <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.tabBar}>
      {options.map((option) => {
        const active = value === option.value;
        return (
          <Pressable key={option.value} onPress={() => onChange(option.value)} style={[styles.tab, active ? styles.tabActive : null]}>
            <Ionicons name={option.icon} size={17} color={active ? "#ffffff" : "#475569"} />
            <Text style={[styles.tabText, active ? styles.tabTextActive : null]}>{option.label}</Text>
          </Pressable>
        );
      })}
    </ScrollView>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <View style={styles.section}>
      <Text style={styles.sectionTitle}>{title}</Text>
      <View style={styles.stack}>{children}</View>
    </View>
  );
}

function ScreenHeading({ title, subtitle }: { title: string; subtitle: string }) {
  return (
    <View>
      <Text style={styles.screenTitle}>{title}</Text>
      <Text style={styles.screenCopy}>{subtitle}</Text>
    </View>
  );
}

function InfoCard({ icon, title, body }: { icon: keyof typeof Ionicons.glyphMap; title: string; body: string }) {
  return (
    <View style={styles.infoCard}>
      <Ionicons name={icon} size={22} color="#0f766e" />
      <Text style={styles.cardTitle}>{title}</Text>
      <Text style={styles.cardBody}>{body}</Text>
    </View>
  );
}

function LoadingCard() {
  return (
    <View style={styles.card}>
      <ActivityIndicator />
      <Text style={styles.muted}>Loading backend features...</Text>
    </View>
  );
}

function EmptyText({ text }: { text: string }) {
  return <Text style={styles.emptyText}>{text}</Text>;
}

function Feedback({ error, notice }: { error?: string; notice?: string }) {
  if (!error && !notice) return null;
  return (
    <View style={[styles.feedback, error ? styles.feedbackError : styles.feedbackNotice]}>
      <Text style={error ? styles.errorText : styles.noticeText}>{error || notice}</Text>
    </View>
  );
}

function callNumber(phone?: string) {
  if (!phone) {
    Alert.alert("Voice phone unavailable", "The backend did not return a voice support number.");
    return;
  }
  Linking.openURL(`tel:${phone.replace(/[^\d+]/g, "")}`).catch(() => {
    Alert.alert("Unable to open phone dialer");
  });
}

function parseLocationCommand(payload: unknown): { command: string; sessionID?: string } | null {
  try {
    const data = typeof payload === "string" ? JSON.parse(payload) : payload;
    if (!data || typeof data !== "object") return null;
    const record = data as Record<string, unknown>;
    const command = String(record.command ?? record.Command ?? "");
    const sessionID = String(record.sessionID ?? record.SessionID ?? "");
    if (!command) return null;
    return { command, sessionID };
  } catch {
    return null;
  }
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: "#f5faf8"
  },
  shell: {
    flex: 1,
    backgroundColor: "#f5faf8"
  },
  center: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: 12
  },
  header: {
    paddingHorizontal: 18,
    paddingVertical: 13,
    borderBottomWidth: 1,
    borderBottomColor: "#d8e6e2",
    backgroundColor: "#ffffff",
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 12
  },
  headerBrand: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    flexShrink: 1
  },
  brandMark: {
    width: 38,
    height: 38,
    borderRadius: 8,
    backgroundColor: "#0f766e",
    alignItems: "center",
    justifyContent: "center"
  },
  brand: {
    fontSize: 19,
    fontWeight: "800",
    color: "#0f172a"
  },
  headerSub: {
    marginTop: 2,
    fontSize: 12,
    color: "#64748b"
  },
  headerActions: {
    flexShrink: 1,
    alignItems: "flex-end"
  },
  authScroll: {
    flexGrow: 1,
    padding: 18,
    justifyContent: "center"
  },
  authPanel: {
    borderRadius: 8,
    borderWidth: 1,
    borderColor: "#d8e6e2",
    backgroundColor: "#ffffff",
    padding: 18,
    gap: 16,
    shadowColor: "#0f766e",
    shadowOpacity: 0.06,
    shadowRadius: 16,
    shadowOffset: { width: 0, height: 8 },
    elevation: 2
  },
  portal: {
    flex: 1
  },
  content: {
    padding: 18,
    paddingBottom: 28
  },
  stack: {
    gap: 12
  },
  screenTitle: {
    fontSize: 24,
    fontWeight: "800",
    color: "#0f172a"
  },
  largeTitle: {
    fontSize: 30,
    fontWeight: "800",
    color: "#ffffff",
    marginTop: 4
  },
  screenCopy: {
    marginTop: 6,
    color: "#475569",
    lineHeight: 21
  },
  heroPanel: {
    borderRadius: 8,
    padding: 20,
    backgroundColor: "#0f766e",
    gap: 12,
    borderWidth: 1,
    borderColor: "#0d9488"
  },
  eyebrow: {
    color: "#ccfbf1",
    fontSize: 12,
    fontWeight: "700",
    textTransform: "uppercase"
  },
  heroCopy: {
    color: "#ecfeff",
    lineHeight: 21
  },
  heroActions: {
    flexDirection: "row",
    gap: 10,
    flexWrap: "wrap"
  },
  gridTwo: {
    gap: 12
  },
  infoCard: {
    borderRadius: 8,
    borderWidth: 1,
    borderColor: "#d8e6e2",
    backgroundColor: "#ffffff",
    padding: 16,
    gap: 8
  },
  card: {
    borderRadius: 8,
    borderWidth: 1,
    borderColor: "#d8e6e2",
    backgroundColor: "#ffffff",
    padding: 14,
    gap: 8
  },
  subCard: {
    borderRadius: 8,
    backgroundColor: "#f5faf8",
    borderWidth: 1,
    borderColor: "#d8e6e2",
    padding: 12,
    gap: 6
  },
  section: {
    gap: 10
  },
  sectionTitle: {
    fontSize: 17,
    fontWeight: "800",
    color: "#0f172a"
  },
  cardTitle: {
    fontSize: 15,
    fontWeight: "800",
    color: "#0f172a",
    textTransform: "capitalize"
  },
  cardBody: {
    color: "#475569",
    lineHeight: 20
  },
  bodyText: {
    color: "#334155",
    lineHeight: 22
  },
  meta: {
    color: "#64748b",
    fontSize: 12,
    lineHeight: 17
  },
  muted: {
    color: "#64748b"
  },
  emptyText: {
    color: "#64748b",
    paddingVertical: 6
  },
  rowBetween: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 12
  },
  flex: {
    flex: 1
  },
  field: {
    gap: 7
  },
  label: {
    color: "#334155",
    fontWeight: "700",
    fontSize: 13
  },
  input: {
    minHeight: 46,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: "#b8c9c5",
    backgroundColor: "#ffffff",
    paddingHorizontal: 12,
    color: "#0f172a",
    fontSize: 16
  },
  textArea: {
    minHeight: 110,
    textAlignVertical: "top",
    paddingTop: 12
  },
  primaryButton: {
    minHeight: 48,
    borderRadius: 8,
    paddingHorizontal: 16,
    backgroundColor: "#0f766e",
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: 8
  },
  primaryText: {
    color: "#ffffff",
    fontWeight: "800",
    fontSize: 15
  },
  disabled: {
    opacity: 0.6
  },
  iconButton: {
    minHeight: 38,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: "#b8c9c5",
    backgroundColor: "#ffffff",
    paddingHorizontal: 11,
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: 6
  },
  iconAccent: {
    borderColor: "#0f766e",
    backgroundColor: "#0f766e"
  },
  iconButtonText: {
    color: "#334155",
    fontWeight: "700"
  },
  iconAccentText: {
    color: "#ffffff"
  },
  textButton: {
    paddingVertical: 6
  },
  textButtonLabel: {
    color: "#0f766e",
    fontWeight: "800"
  },
  inlineActions: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 14
  },
  segmented: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 6,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: "#d8e6e2",
    backgroundColor: "#f5faf8",
    padding: 4
  },
  segment: {
    minHeight: 34,
    borderRadius: 7,
    paddingHorizontal: 10,
    alignItems: "center",
    justifyContent: "center"
  },
  segmentActive: {
    backgroundColor: "#0f766e"
  },
  segmentText: {
    color: "#475569",
    fontWeight: "700",
    textTransform: "capitalize"
  },
  segmentTextActive: {
    color: "#ffffff"
  },
  tabBar: {
    paddingHorizontal: 12,
    paddingVertical: 10,
    gap: 8,
    borderBottomWidth: 1,
    borderBottomColor: "#d8e6e2",
    backgroundColor: "#ffffff"
  },
  tab: {
    minHeight: 38,
    borderRadius: 8,
    paddingHorizontal: 12,
    borderWidth: 1,
    borderColor: "#d8e6e2",
    backgroundColor: "#ffffff",
    flexDirection: "row",
    alignItems: "center",
    gap: 6
  },
  tabActive: {
    borderColor: "#0f766e",
    backgroundColor: "#0f766e"
  },
  tabText: {
    color: "#475569",
    fontWeight: "800"
  },
  tabTextActive: {
    color: "#ffffff"
  },
  badge: {
    overflow: "hidden",
    borderRadius: 999,
    backgroundColor: "#dcfce7",
    color: "#166534",
    paddingHorizontal: 9,
    paddingVertical: Platform.select({ ios: 4, default: 3 }),
    fontSize: 12,
    fontWeight: "800",
    textTransform: "capitalize"
  },
  badgeWarn: {
    backgroundColor: "#fee2e2",
    color: "#991b1b"
  },
  switchTrack: {
    width: 52,
    height: 30,
    borderRadius: 999,
    padding: 3,
    justifyContent: "center"
  },
  switchOn: {
    backgroundColor: "#0f766e"
  },
  switchOff: {
    backgroundColor: "#cbd5e1"
  },
  switchThumb: {
    width: 24,
    height: 24,
    borderRadius: 999,
    backgroundColor: "#ffffff"
  },
  switchThumbOn: {
    alignSelf: "flex-end"
  },
  switchThumbOff: {
    alignSelf: "flex-start"
  },
  feedback: {
    borderRadius: 8,
    padding: 12,
    borderWidth: 1
  },
  feedbackError: {
    backgroundColor: "#fef2f2",
    borderColor: "#fecaca"
  },
  feedbackNotice: {
    backgroundColor: "#ecfdf5",
    borderColor: "#bbf7d0"
  },
  errorText: {
    color: "#991b1b",
    lineHeight: 19
  },
  noticeText: {
    color: "#166534",
    lineHeight: 19
  }
});
