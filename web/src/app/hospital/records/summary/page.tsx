"use client";

export const dynamic = "force-dynamic";

import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import Image from "next/image";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import QRCode from "qrcode";
import { Room, RoomEvent, Track, type RemoteTrack } from "livekit-client";
import { Button } from "../../../../components/ui/button";
import { Input } from "../../../../components/ui/input";
import { type SidebarItem } from "../../../../components/ui/sidebar";
import { StatCard } from "../../../../components/ui/stat-card";
import { DashboardShell } from "../../../../components/layout/dashboard-shell";
import { PageHeader } from "../../../../components/layout/page-header";
import { StatusBanner } from "../../../../components/status-banner";
import { toast } from "sonner";
import {
  APIEndpoints,
  AuditEventRecord,
  BridgedCallSessionRecord,
  ConnectBridgedCallRequest,
  EndBridgedCallRequest,
  getAuditEventTypeOptions,
  HospitalIncidentRecord,
  HospitalConsentRequestRecord,
  HospitalMeetingRecord,
  HospitalPatientRecord,
  HTTPBridgedCallConnectResponse,
  HTTPBridgedCallSessionListResponse,
  HTTPBridgedCallSessionResponse,
  InterpretationSegmentMessage,
  HTTPHospitalConsentRequestCreateRequest,
  HTTPHospitalConsentRequestCreateResponse,
  HTTPHospitalConsentRequestListResponse,
  HTTPHospitalIncidentListResponse,
  HTTPHospitalMeetingListResponse,
  HTTPHospitalMeetingMutationResponse,
  HTTPHospitalMeetingRescheduleRequest,
  HTTPHospitalPatientAuditResponse,
  HTTPHospitalPatientListResponse,
  HTTPHospitalPatientSummaryRequest,
  HTTPHospitalPatientSummaryResponse,
  HTTPHospitalStaffRegisterRequest,
  HTTPHospitalStaffRegisterResponse,
  UpdateBridgedCallTranslationRequest,
} from "../../../../contracts";
import { apiFetch, cachedApiJSON, clearApiCache, logoutAuth, preloadApiJSON } from "../../../../lib/auth-client";
import { useAuth } from "../../../../hooks/useAuth";
import {
  Activity,
  AlertCircle,
  CalendarDays,
  CheckCircle,
  Clock,
  ExternalLink,
  FileSearch,
  FileText,
  Home,
  ListChecks,
  QrCode,
  RefreshCw,
  Search,
  Shield,
  Sparkles,
  UserPlus,
  Languages,
  Video,
  XCircle,
} from "lucide-react";
import { formatDateTime, formatEventType, formatTimeOnly, meaningfulDate } from "../../../../lib/format";
import { resolveIANATimezone } from "../../../../lib/timezone";

const focusOptions = [
  { value: "full", label: "Full Summary" },
  { value: "medications", label: "Medications Only" },
  { value: "allergies", label: "Allergies Only" },
  { value: "diagnoses", label: "Diagnoses Only" },
];

const navItems: SidebarItem[] = [
  { id: "home", label: "Home", icon: Home },
  { id: "summary", label: "Summary", icon: FileText },
  { id: "meetings", label: "Meetings", icon: CalendarDays },
  { id: "staff", label: "Staff", icon: UserPlus },
  { id: "consent", label: "Consent", icon: QrCode },
  { id: "incidents", label: "Incidents", icon: AlertCircle },
  { id: "audit", label: "Audit Search", icon: FileSearch },
];

function formatMeetingStatus(value?: string) {
  return value?.replaceAll("_", " ") || "pending";
}

function looksLikeBridgePatientIdentity(identity?: string) {
  const value = (identity || "").trim().toLowerCase();
  if (!value || value.startsWith("staff-")) return false;
  if (value.startsWith("sip_") || value.startsWith("+")) return true;
  const digits = value.replace(/\D/g, "");
  return digits.length >= 10;
}

function bridgeCaptionLabel(participant?: string) {
  return participant === "staff" ? "Clinician" : "Patient";
}

function hospitalMeetingPath(meetingID: string, action?: string) {
  const base = `${APIEndpoints.HOSPITAL_MEETINGS}/${encodeURIComponent(meetingID)}`;
  return action ? `${base}/${action}` : base;
}

function toDateTimeLocalValue(value?: string) {
  const date = meaningfulDate(value);
  if (!date) return "";
  const offset = date.getTimezoneOffset();
  const local = new Date(date.getTime() - offset * 60_000);
  return local.toISOString().slice(0, 16);
}

function dateTimeLocalToRFC3339(value: string) {
  return new Date(value).toISOString();
}

function filterAuditEvents(
  events: AuditEventRecord[],
  type: string,
  start: string,
  end: string,
) {
  const startTime = start ? new Date(start).getTime() : null;
  const endTime = end ? new Date(end).getTime() : null;

  return events.filter((event) => {
    if (type !== "all" && event.event_type !== type) return false;
    const eventDate = meaningfulDate(event.timestamp);
    if ((startTime || endTime) && !eventDate) return false;
    const eventTime = eventDate?.getTime();
    if (startTime && eventTime !== undefined && eventTime < startTime) return false;
    if (endTime && eventTime !== undefined && eventTime > endTime) return false;
    return true;
  });
}

function HospitalTriageStrip({
  incidents,
  bridges,
  meetings,
  onNavigate,
}: {
  incidents: HospitalIncidentRecord[];
  bridges: BridgedCallSessionRecord[];
  meetings: HospitalMeetingRecord[];
  onNavigate: (section: string) => void;
}) {
  const activeMeetings = meetings.filter((meeting) => meeting.status !== "cancelled");
  return (
    <section className="rounded-[var(--zh-radius-card)] border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-700 dark:bg-slate-900" aria-label="Triage strip">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div><p className="text-base font-bold text-slate-950 dark:text-white">What needs attention now</p><p className="mt-1 text-base text-slate-600 dark:text-slate-300">Triage incidents, incoming bridges, and today&apos;s schedule from one view.</p></div>
        <span className="rounded-full bg-indigo-50 px-3 py-1 text-sm font-bold text-indigo-700">Live clinical workspace</span>
      </div>
      <div className="mt-4 grid gap-3 md:grid-cols-3">
        <button type="button" className="min-h-[72px] rounded-xl border border-rose-200 bg-rose-50 p-4 text-left hover:border-rose-400" onClick={() => onNavigate("incidents")}><span className="text-2xl font-bold text-rose-800">{incidents.length}</span><span className="mt-1 block text-sm font-bold text-rose-900">Open incidents</span></button>
        <button type="button" className="min-h-[72px] rounded-xl border border-amber-200 bg-amber-50 p-4 text-left hover:border-amber-400" onClick={() => onNavigate("home")}><span className="text-2xl font-bold text-amber-800">{bridges.length}</span><span className="mt-1 block text-sm font-bold text-amber-900">Incoming bridge calls</span></button>
        <button type="button" className="min-h-[72px] rounded-xl border border-indigo-200 bg-indigo-50 p-4 text-left hover:border-indigo-400" onClick={() => onNavigate("appointments")}><span className="text-2xl font-bold text-indigo-800">{activeMeetings.length}</span><span className="mt-1 block text-sm font-bold text-indigo-900">Today&apos;s schedule</span></button>
      </div>
    </section>
  );
}

function HospitalPatientSummaryPageContent() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const { ready, authenticated, accessToken, staffRole } = useAuth("hospital");

  useEffect(() => {
    if (ready && !authenticated) {
      router.replace("/login/hospital");
    }
  }, [ready, authenticated, router]);

  const activeSection = searchParams.get("section") || "home";
  const setActiveSection = useCallback(
    (nextSection: string) => {
      const params = new URLSearchParams(searchParams.toString());
      params.set("section", nextSection);
      router.replace(`${pathname}?${params.toString()}`, { scroll: false });
    },
    [pathname, router, searchParams],
  );
  const sectionNavItems = useMemo(
    () =>
      navItems.map((item) => ({
        ...item,
        href: `${pathname}?section=${item.id}`,
      })),
    [pathname],
  );
  const [patientID, setPatientID] = useState("");
  const [patientSearch, setPatientSearch] = useState("");
  const [patients, setPatients] = useState<HospitalPatientRecord[]>([]);
  const [focus, setFocus] = useState("full");
  const [summary, setSummary] = useState("");
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [isRefreshingIncidents, setIsRefreshingIncidents] = useState(false);
  const [isRefreshingMeetings, setIsRefreshingMeetings] = useState(false);
  const [isLoadingPatients, setIsLoadingPatients] = useState(false);
  const [isRefreshingConsentRequests, setIsRefreshingConsentRequests] = useState(false);
  const [mutatingMeetingID, setMutatingMeetingID] = useState<string | null>(null);
  const [incidents, setIncidents] = useState<HospitalIncidentRecord[]>([]);
  const [meetings, setMeetings] = useState<HospitalMeetingRecord[]>([]);
  const [consentRequests, setConsentRequests] = useState<HospitalConsentRequestRecord[]>([]);
  const [auditEvents, setAuditEvents] = useState<AuditEventRecord[]>([]);
  const [staffNotice, setStaffNotice] = useState("");
  const [showScheduleForm, setShowScheduleForm] = useState(false);
  const [scheduleForm, setScheduleForm] = useState({
    patient_id: "",
    starts_at: "",
    duration_minutes: 30,
    timezone: resolveIANATimezone(),
    title: "",
    notes: "",
  });
  const [submittingSchedule, setSubmittingSchedule] = useState(false);
  const [consentQRCode, setConsentQRCode] = useState("");
  const [bridgedSession, setBridgedSession] = useState<BridgedCallSessionRecord | null>(null);
  const [bridgedForm, setBridgedForm] = useState({
    session_id: "",
    staff_participant_identity: "",
    translation_enabled: false,
    language_mode: "auto",
    language_code: "en",
  });
  const [pendingBridges, setPendingBridges] = useState<BridgedCallSessionRecord[]>([]);
  const [isRefreshingBridges, setIsRefreshingBridges] = useState(false);
  const [bridgeCaptions, setBridgeCaptions] = useState<InterpretationSegmentMessage[]>([]);
  const [bridgeRoomState, setBridgeRoomState] = useState<"disconnected" | "connecting" | "connected">("disconnected");
  const bridgeRoomRef = useRef<Room | null>(null);
  const bridgeAudioRef = useRef<HTMLDivElement | null>(null);

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
    if (bridgeAudioRef.current) bridgeAudioRef.current.innerHTML = "";
  }, []);

  useEffect(() => {
    return () => {
      void disconnectBridgeRoom();
    };
  }, [disconnectBridgeRoom]);

  const joinBridgeRoom = useCallback(
    async (wsUrl: string, token: string) => {
      if (!wsUrl.startsWith("ws://") && !wsUrl.startsWith("wss://")) {
        setError("LiveKit URL unavailable for this bridge; captions and audio disabled.");
        return;
      }
      await disconnectBridgeRoom();
      const room = new Room();
      bridgeRoomRef.current = room;
      setBridgeRoomState("connecting");
      room.on(RoomEvent.DataReceived, (payload: Uint8Array, _participant, _kind, topic?: string) => {
        if (topic !== "zorba.interpretation") return;
        try {
          const message: InterpretationSegmentMessage = JSON.parse(new TextDecoder().decode(payload));
          if (message.type !== "interpretation.segment") return;
          setBridgeCaptions((current) => [...current.slice(-49), message]);
        } catch {
          // Ignore malformed data packets.
        }
      });
      room.on(RoomEvent.TrackSubscribed, (track: RemoteTrack, _publication, participant) => {
        if (
          track.kind === Track.Kind.Audio &&
          bridgeAudioRef.current &&
          looksLikeBridgePatientIdentity(participant?.identity)
        ) {
          bridgeAudioRef.current.appendChild(track.attach());
        }
      });
      room.on(RoomEvent.Disconnected, () => {
        if (bridgeRoomRef.current === room) {
          bridgeRoomRef.current = null;
          setBridgeRoomState("disconnected");
        }
      });
      try {
        await room.connect(wsUrl, token);
        await room.localParticipant.setMicrophoneEnabled(true).catch(() => {
          // Listen-only when mic permission is denied.
        });
        setBridgeRoomState("connected");
      } catch {
        setError("Bridge connected, but the LiveKit room join failed. Captions/audio unavailable.");
        await disconnectBridgeRoom();
      }
    },
    [disconnectBridgeRoom],
  );

  const loadPendingBridges = useCallback(
    async (isRefresh = false) => {
      if (!accessToken) return;
      if (isRefresh) setIsRefreshingBridges(true);
      try {
        const data = await cachedApiJSON<HTTPBridgedCallSessionListResponse>(
          "hospital",
          `${APIEndpoints.HOSPITAL_BRIDGED_CALL_SESSIONS}?status=transfer_requested`,
          { ttlMs: 15_000, force: isRefresh },
        );
        setPendingBridges(data.data?.sessions ?? []);
      } catch {
        // Best-effort side panel.
      } finally {
        setIsRefreshingBridges(false);
      }
    },
    [accessToken],
  );

  useEffect(() => {
    void loadPendingBridges();
  }, [loadPendingBridges]);

  useEffect(() => {
    if (!accessToken) return;
    const timer = window.setInterval(() => {
      void loadPendingBridges(true);
    }, 8_000);
    return () => window.clearInterval(timer);
  }, [accessToken, loadPendingBridges]);

  const loadPatients = useCallback(
    async (query: string, force = false) => {
      if (!accessToken) return;
      setIsLoadingPatients(true);
      try {
        const suffix = query.trim()
          ? `?query=${encodeURIComponent(query.trim())}`
          : "";
        const data = await cachedApiJSON<HTTPHospitalPatientListResponse>(
          "hospital",
          `${APIEndpoints.HOSPITAL_PATIENTS}${suffix}`,
          { ttlMs: 60_000, force: force || Boolean(query.trim()) },
        );
        setPatients(data.data?.patients ?? []);
      } catch {
        setError("Network error while loading consented patients.");
      } finally {
        setIsLoadingPatients(false);
      }
    },
    [accessToken],
  );

  useEffect(() => {
    void loadPatients("");
  }, [loadPatients]);

  const loadIncidents = useCallback(
    async (isRefresh = false) => {
      if (!accessToken) return;
      if (isRefresh) setIsRefreshingIncidents(true);
      try {
        const data = await cachedApiJSON<HTTPHospitalIncidentListResponse>(
          "hospital",
          APIEndpoints.HOSPITAL_INCIDENTS,
          { ttlMs: 30_000, force: isRefresh },
        );
        setIncidents(data.data?.incidents ?? []);
      } catch {
        // Best-effort side panel.
      } finally {
        setIsRefreshingIncidents(false);
      }
    },
    [accessToken],
  );

  useEffect(() => {
    void loadIncidents();
  }, [loadIncidents]);

  const loadMeetings = useCallback(
    async (isRefresh = false) => {
      if (!accessToken) return;
      if (isRefresh) setIsRefreshingMeetings(true);
      try {
        const data = await cachedApiJSON<HTTPHospitalMeetingListResponse>(
          "hospital",
          APIEndpoints.HOSPITAL_MEETINGS,
          { ttlMs: 45_000, force: isRefresh },
        );
        setMeetings(data.data?.meetings ?? []);
      } catch {
        // Best-effort dashboard panel.
      } finally {
        setIsRefreshingMeetings(false);
      }
    },
    [accessToken],
  );

  useEffect(() => {
    void loadMeetings();
  }, [loadMeetings]);

  const loadConsentRequests = useCallback(
    async (isRefresh = false) => {
      if (!accessToken) return;
      if (isRefresh) setIsRefreshingConsentRequests(true);
      try {
        const data = await cachedApiJSON<HTTPHospitalConsentRequestListResponse>(
          "hospital",
          APIEndpoints.HOSPITAL_CONSENT_REQUESTS,
          { ttlMs: 45_000, force: isRefresh },
        );
        setConsentRequests(data.data?.requests ?? []);
      } catch {
        // Best-effort consent panel.
      } finally {
        setIsRefreshingConsentRequests(false);
      }
    },
    [accessToken],
  );

  useEffect(() => {
    void loadConsentRequests();
  }, [loadConsentRequests]);

  useEffect(() => {
    if (!ready || !authenticated || !accessToken) return;
    preloadApiJSON<HTTPHospitalIncidentListResponse>("hospital", APIEndpoints.HOSPITAL_INCIDENTS, { ttlMs: 30_000 });
    preloadApiJSON<HTTPHospitalMeetingListResponse>("hospital", APIEndpoints.HOSPITAL_MEETINGS, { ttlMs: 45_000 });
    preloadApiJSON<HTTPHospitalConsentRequestListResponse>("hospital", APIEndpoints.HOSPITAL_CONSENT_REQUESTS, { ttlMs: 45_000 });
    preloadApiJSON<HTTPBridgedCallSessionListResponse>(
      "hospital",
      `${APIEndpoints.HOSPITAL_BRIDGED_CALL_SESSIONS}?status=transfer_requested`,
      { ttlMs: 15_000 },
    );
  }, [accessToken, authenticated, ready]);

  const mergeMeeting = (meeting?: HospitalMeetingRecord) => {
    if (!meeting?.id) return;
    clearApiCache("hospital");
    setMeetings((current) => {
      const exists = current.some((item) => item.id === meeting.id);
      if (!exists) return [meeting, ...current];
      return current.map((item) => (item.id === meeting.id ? meeting : item));
    });
  };

  const handleAcceptMeeting = async (meetingID: string) => {
    setMutatingMeetingID(meetingID);
    setError("");
    try {
      const response = await apiFetch("hospital", hospitalMeetingPath(meetingID, "accept"), {
        method: "POST",
      });
      const data: HTTPHospitalMeetingMutationResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to accept meeting request.");
        return;
      }
      mergeMeeting(data.data?.meeting);
    } catch {
      setError("Network error while accepting meeting request.");
    } finally {
      setMutatingMeetingID(null);
    }
  };

  const handleRescheduleMeeting = async (
    meetingID: string,
    payload: HTTPHospitalMeetingRescheduleRequest,
  ) => {
    setMutatingMeetingID(meetingID);
    setError("");
    try {
      const response = await apiFetch("hospital", hospitalMeetingPath(meetingID, "reschedule"), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data: HTTPHospitalMeetingMutationResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to reschedule meeting request.");
        return;
      }
      mergeMeeting(data.data?.meeting);
    } catch {
      setError("Network error while rescheduling meeting request.");
    } finally {
      setMutatingMeetingID(null);
    }
  };

  const handleCancelMeeting = async (meetingID: string) => {
    setMutatingMeetingID(meetingID);
    setError("");
    try {
      const response = await apiFetch("hospital", hospitalMeetingPath(meetingID), {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reason: "Cancelled from hospital dashboard" }),
      });
      const data: HTTPHospitalMeetingMutationResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to cancel meeting request.");
        return;
      }
      mergeMeeting(data.data?.meeting);
    } catch {
      setError("Network error while cancelling meeting request.");
    } finally {
      setMutatingMeetingID(null);
    }
  };

  const connectBridgedCall = async (sessionIdOverride?: string, joinMode: "web" | "phone" = "web") => {
    const sessionId = (sessionIdOverride ?? bridgedForm.session_id).trim();
    if (!accessToken || !sessionId) return;
    setError("");
    try {
      const translationSaved = await updateBridgedTranslationForSession(sessionId);
      if (!translationSaved) return;
      const payload: ConnectBridgedCallRequest = {
        session_id: sessionId,
        staff_participant_identity: bridgedForm.staff_participant_identity.trim() || undefined,
        join_mode: joinMode,
      };
      const response = await apiFetch("hospital", APIEndpoints.HOSPITAL_BRIDGED_CALL_CONNECT, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data: HTTPBridgedCallConnectResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to connect bridged call.");
        return;
      }
      setBridgedSession(data.data?.session ?? null);
      setBridgeCaptions([]);
      clearApiCache("hospital");
      void loadPendingBridges(true);
      if (joinMode === "phone") {
        setError("");
        return;
      }
      const token = data.data?.staff_room_token;
      const wsUrl = data.data?.livekit_ws_url;
      if (token && wsUrl) {
        await joinBridgeRoom(wsUrl, token);
      }
    } catch {
      setError("Network error while connecting bridged call.");
    }
  };

  const refreshBridgedSession = async () => {
    if (!accessToken || !bridgedForm.session_id.trim()) return;
    setError("");
    try {
      const response = await apiFetch(
        "hospital",
        `${APIEndpoints.HOSPITAL_BRIDGED_CALL_SESSION}?session_id=${encodeURIComponent(bridgedForm.session_id.trim())}`,
      );
      const data: HTTPBridgedCallSessionResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to load bridged call session.");
        return;
      }
      setBridgedSession(data.data?.session ?? null);
    } catch {
      setError("Network error while loading bridged session.");
    }
  };

  const updateBridgedTranslation = async () => {
    await updateBridgedTranslationForSession(bridgedForm.session_id.trim());
  };

  const updateBridgedTranslationForSession = async (sessionID: string) => {
    if (!accessToken || !sessionID.trim()) return false;
    setError("");
    try {
      const payload: UpdateBridgedCallTranslationRequest = {
        session_id: sessionID.trim(),
        participant: "staff",
        translation: {
          enabled: bridgedForm.translation_enabled,
          language_mode: bridgedForm.language_mode,
          language_code: bridgedForm.language_code.trim().toLowerCase(),
          participant_identity: bridgedForm.staff_participant_identity.trim() || undefined,
        },
      };
      const response = await apiFetch("hospital", APIEndpoints.HOSPITAL_BRIDGED_CALL_TRANSLATION, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data: HTTPBridgedCallSessionResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to update translation preferences.");
        return false;
      }
      setBridgedSession(data.data?.session ?? null);
      return true;
    } catch {
      setError("Network error while updating translation preferences.");
      return false;
    }
  };

  const endBridgedCall = async () => {
    if (!accessToken || !bridgedForm.session_id.trim()) return;
    setError("");
    try {
      const payload: EndBridgedCallRequest = {
        session_id: bridgedForm.session_id.trim(),
        reason: "Ended from hospital dashboard",
      };
      const response = await apiFetch("hospital", APIEndpoints.HOSPITAL_BRIDGED_CALL_END, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data: HTTPBridgedCallSessionResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to end bridged call.");
        return;
      }
      setBridgedSession(data.data?.session ?? null);
      await disconnectBridgeRoom();
      clearApiCache("hospital");
      void loadPendingBridges(true);
    } catch {
      setError("Network error while ending bridged call.");
    }
  };

  const handleScheduleMeeting = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!accessToken) return;
    const timezone = resolveIANATimezone(scheduleForm.timezone);
    if (timezone !== scheduleForm.timezone) {
      setScheduleForm((prev) => ({ ...prev, timezone }));
    }
    setSubmittingSchedule(true);
    try {
      const response = await apiFetch("hospital", APIEndpoints.HOSPITAL_MEETINGS, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          patient_id: scheduleForm.patient_id.trim(),
          starts_at: new Date(scheduleForm.starts_at).toISOString(),
          duration_minutes: scheduleForm.duration_minutes,
          timezone,
          title: scheduleForm.title || undefined,
          notes: scheduleForm.notes || undefined,
        }),
      });
      const data: HTTPHospitalMeetingMutationResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to schedule meeting.");
        return;
      }
      if (data.data?.meeting) {
        setMeetings((prev) => [data.data!.meeting!, ...prev]);
      }
      clearApiCache("hospital");
      setShowScheduleForm(false);
      setScheduleForm((prev) => ({ ...prev, patient_id: "", starts_at: "", title: "", notes: "" }));
    } catch {
      setError("Network error while scheduling meeting.");
    } finally {
      setSubmittingSchedule(false);
    }
  };

  const handleRegisterStaff = async (payload: HTTPHospitalStaffRegisterRequest) => {
    setError("");
    setStaffNotice("");
    try {
      const response = await apiFetch("hospital", APIEndpoints.HOSPITAL_STAFF_REGISTER, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data: HTTPHospitalStaffRegisterResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to register staff.");
        return false;
      }
      setStaffNotice(
        `Staff account created${data.data?.staff_id ? ` (${data.data.staff_id})` : ""}.`,
      );
      return true;
    } catch {
      setError("Network error while registering staff.");
      return false;
    }
  };

  const handleCreateConsentRequest = async (
    payload: HTTPHospitalConsentRequestCreateRequest,
  ) => {
    setError("");
    try {
      const response = await apiFetch("hospital", APIEndpoints.HOSPITAL_CONSENT_REQUESTS, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data: HTTPHospitalConsentRequestCreateResponse = await response.json();
      if (!response.ok || !data.data?.request) {
        setError(data.error?.message || "Unable to create consent request.");
        return false;
      }
      const request = data.data.request;
      setConsentRequests((current) => [request, ...current]);
      setConsentQRCode(await QRCode.toDataURL(request.qr_payload || request.token || ""));
      clearApiCache("hospital");
      return true;
    } catch {
      setError("Network error while creating consent request.");
      return false;
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setSummary("");

    if (!accessToken) {
      setError("Please log in as hospital staff first.");
      return;
    }

    setIsLoading(true);
    const payload: HTTPHospitalPatientSummaryRequest = {
      patient_id: patientID.trim(),
      focus,
    };

    try {
      const response = await apiFetch("hospital", APIEndpoints.HOSPITAL_PATIENT_SUMMARY, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data: HTTPHospitalPatientSummaryResponse = await response.json();

      if (!response.ok) {
        setError(data.error?.message || "Failed to summarize patient records.");
        return;
      }

      setPatientID(data.data?.patient_id || payload.patient_id);
      setSummary(data.data?.summary || "No summary was returned.");
      const auditResponse = await apiFetch(
        "hospital",
        `${APIEndpoints.HOSPITAL_PATIENT_AUDIT}?patient_id=${encodeURIComponent(data.data?.patient_id || payload.patient_id)}`,
      );
      const auditData: HTTPHospitalPatientAuditResponse =
        await auditResponse.json();
      setAuditEvents(auditResponse.ok ? auditData.data?.events ?? [] : []);
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setIsLoading(false);
    }
  };

  return (
      <DashboardShell
        title="Hospital Console"
        subtitle="Clinical record command center"
        navItems={sectionNavItems}
        activeSection={activeSection}
        onSectionChange={setActiveSection}
        onLogout={() => {
          void logoutAuth("hospital").then(() => router.push("/login/hospital"));
        }}
      >
        <PageHeader
          eyebrow="Hospital Console"
          title="Clinical Record Manager"
          description="Summarize medical files, inspect access trails, and monitor emergency voice alerts in real time."
          actions={
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                toast.message("Refreshing clinical workspace...");
                void Promise.all([loadPatients(patientSearch, true), loadMeetings(true), loadIncidents(true)]);
              }}
            >
              <RefreshCw className="h-4 w-4" />
              Refresh workspace
            </Button>
          }
        />

          <div className="grid gap-4 md:grid-cols-4">
            <StatCard
              icon={FileText}
              label="Summary Status"
              value={summary ? "Ready" : "Pending"}
              trend={patientID || "No patient selected"}
            />
            <StatCard
              icon={AlertCircle}
              label="Open Incidents"
              value={incidents.length}
              tone="rose"
              trend="Voice safety feed"
            />
            <StatCard
              icon={CalendarDays}
              label="Visit Requests"
              value={meetings.filter((meeting) => meeting.status !== "cancelled").length}
              tone="orange"
              trend="Dashboard scheduling"
            />
            <StatCard
              icon={Shield}
              label="Audit Events"
              value={auditEvents.length}
              tone="emerald"
              trend="Last patient lookup"
            />
          </div>
          <HospitalTriageStrip incidents={incidents} bridges={pendingBridges} meetings={meetings} onNavigate={setActiveSection} />

          {error ? <StatusBanner tone="error" message={error} /> : null}

          {activeSection === "home" ? (
            <div className="space-y-6">
              <PatientHomePanel
                patients={patients}
                loading={isLoadingPatients}
                query={patientSearch}
                onQueryChange={setPatientSearch}
                onSearch={() => void loadPatients(patientSearch)}
                onRefresh={() => void loadPatients("", true)}
                onSelectSummary={(patient) => {
                  setPatientID(patient.patient_id || "");
                  setActiveSection("summary");
                }}
                onSelectAudit={(patient) => {
                  setPatientID(patient.patient_id || "");
                  setActiveSection("audit");
                }}
              />
              <section className="clinical-card p-6">
                <h2 className="flex items-center gap-2 text-lg font-black">
                  <Languages className="h-5 w-5 text-indigo-600" />
                  Bridged Call Interpretation
                </h2>
                <p className="mt-2 text-sm leading-7 text-slate-600">
                  Connect to a transferred patient call, decide how the translated audio should be spoken to staff, and close the bridge when the consult ends.
                </p>
                <div className="mt-5 rounded-2xl border border-slate-200 bg-slate-50/70 p-4">
                  <div className="flex items-center justify-between">
                    <p className="text-sm font-black text-slate-900">Incoming bridged calls</p>
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={() => void loadPendingBridges(true)}
                    >
                      <RefreshCw className={`h-4 w-4 ${isRefreshingBridges ? "animate-spin" : ""}`} />
                    </Button>
                  </div>
                  {pendingBridges.length === 0 ? (
                    <p className="mt-2 text-sm text-slate-500">
                      No patients are waiting for a bridged consult right now.
                    </p>
                  ) : (
                    <ul className="mt-3 space-y-3">
                      {pendingBridges.map((session) => (
                        <li
                          key={session.session_id}
                          className="rounded-2xl border-2 border-amber-300 bg-amber-50 px-4 py-3 text-sm shadow-sm animate-pulse"
                        >
                          <div className="flex flex-wrap items-start justify-between gap-3">
                            <div>
                              <p className="text-xs font-black uppercase tracking-wide text-amber-800">
                                Ringing — patient requesting staff
                              </p>
                              <p className="mt-1 font-bold text-slate-900">{session.session_id}</p>
                              <p className="text-xs text-slate-600">
                                Patient {session.patient_id || "unknown"}
                                {session.transfer_reason ? ` — ${session.transfer_reason}` : ""}
                              </p>
                              <p className="mt-2 text-xs text-slate-500">
                                Set your listen language below, then accept in browser or by phone.
                              </p>
                            </div>
                            <div className="flex flex-wrap gap-2">
                              <Button
                                type="button"
                                variant="healthcare"
                                size="sm"
                                onClick={() => {
                                  setBridgedForm((current) => ({
                                    ...current,
                                    session_id: session.session_id || "",
                                    translation_enabled: true,
                                  }));
                                  void connectBridgedCall(session.session_id, "web");
                                }}
                              >
                                Accept (Web)
                              </Button>
                              <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={() => {
                                  setBridgedForm((current) => ({
                                    ...current,
                                    session_id: session.session_id || "",
                                    translation_enabled: true,
                                  }));
                                  void connectBridgedCall(session.session_id, "phone");
                                }}
                              >
                                Accept (Phone)
                              </Button>
                            </div>
                          </div>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
                <div className="mt-5 grid gap-4 md:grid-cols-2">
                  <Input
                    label="Session ID"
                    value={bridgedForm.session_id}
                    onChange={(event) =>
                      setBridgedForm((current) => ({
                        ...current,
                        session_id: event.target.value,
                      }))
                    }
                    placeholder="voice session or room SID"
                  />
                  <Input
                    label="Staff Participant Identity"
                    value={bridgedForm.staff_participant_identity}
                    onChange={(event) =>
                      setBridgedForm((current) => ({
                        ...current,
                        staff_participant_identity: event.target.value,
                      }))
                    }
                    placeholder="Optional"
                  />
                  <Input
                    label="Listen in Language"
                    value={bridgedForm.language_code}
                    onChange={(event) =>
                      setBridgedForm((current) => ({
                        ...current,
                        language_code: event.target.value,
                      }))
                    }
                    placeholder="en"
                  />
                </div>
                <div className="mt-4 flex flex-wrap items-center gap-3">
                  <button
                    type="button"
                    className={`rounded-full px-4 py-2 text-sm font-bold ${bridgedForm.translation_enabled ? "bg-indigo-600 text-white" : "bg-slate-100 text-slate-700"}`}
                    onClick={() =>
                      setBridgedForm((current) => ({
                        ...current,
                        translation_enabled: !current.translation_enabled,
                      }))
                    }
                  >
                    {bridgedForm.translation_enabled ? "Translation On" : "Translation Off"}
                  </button>
                  <button
                    type="button"
                    className={`rounded-full px-4 py-2 text-sm font-bold ${bridgedForm.language_mode === "auto" ? "bg-emerald-600 text-white" : "bg-slate-100 text-slate-700"}`}
                    onClick={() =>
                      setBridgedForm((current) => ({
                        ...current,
                        language_mode: current.language_mode === "auto" ? "manual" : "auto",
                      }))
                    }
                  >
                    {bridgedForm.language_mode === "auto" ? "Auto Detect" : "Manual Language"}
                  </button>
                </div>
                <div className="mt-5 flex flex-wrap gap-3">
                  <Button type="button" variant="healthcare" onClick={() => void connectBridgedCall(undefined, "web")}>
                    Connect (Web)
                  </Button>
                  <Button type="button" variant="outline" onClick={() => void connectBridgedCall(undefined, "phone")}>
                    Connect (Phone)
                  </Button>
                  <Button type="button" variant="outline" onClick={() => void refreshBridgedSession()}>
                    Refresh Session
                  </Button>
                  <Button type="button" variant="outline" onClick={() => void updateBridgedTranslation()}>
                    Save Translation
                  </Button>
                  <Button type="button" variant="outline" onClick={() => void endBridgedCall()}>
                    End Bridge
                  </Button>
                </div>
                {bridgedSession ? (
                  <div className="mt-5 rounded-2xl border border-indigo-100 bg-indigo-50/70 p-4 text-sm text-slate-700">
                    <p className="font-black text-indigo-950">
                      Session status: {bridgedSession.status || "pending"}
                    </p>
                    <p className="mt-2">Patient: {bridgedSession.patient_id || "unknown"}</p>
                    <p>Hospital: {bridgedSession.hospital_id || "unknown"}</p>
                    <p>Staff: {bridgedSession.staff_id || "awaiting assignment"}</p>
                    <p>
                      Staff translation: {bridgedSession.staff_translation?.enabled ? "enabled" : "disabled"} / {bridgedSession.staff_translation?.language_mode || "auto"} / {bridgedSession.staff_translation?.language_code || "default"}
                    </p>
                    <p className="mt-2">
                      Live room:{" "}
                      <span className={bridgeRoomState === "connected" ? "font-bold text-emerald-700" : "font-bold text-slate-500"}>
                        {bridgeRoomState}
                      </span>
                    </p>
                    <p className="mt-2">
                      Interpreter mode:{" "}
                      <span className="font-bold text-indigo-900">
                        {bridgedSession.status === "connected" ? "live" : "waiting for clinician audio"}
                      </span>
                    </p>
                  </div>
                ) : null}
                <div ref={bridgeAudioRef} className="hidden" aria-hidden />
                {bridgeRoomState !== "disconnected" || bridgeCaptions.length > 0 ? (
                  <div className="mt-5 rounded-2xl border border-emerald-100 bg-emerald-50/60 p-4">
                    <p className="text-sm font-black text-emerald-950">Live interpretation captions</p>
                    <p className="mt-1 text-xs font-semibold text-emerald-800">
                      Patient audio stays audible here; translated clinician speech is captioned but not replayed to avoid echo.
                    </p>
                    {bridgeCaptions.length === 0 ? (
                      <p className="mt-2 text-sm text-slate-500">
                        Waiting for the patient or clinician to speak...
                      </p>
                    ) : (
                      <ul className="mt-3 max-h-64 space-y-2 overflow-y-auto text-sm">
                        {bridgeCaptions.map((caption, index) => (
                          <li key={index} className="rounded-xl bg-white px-3 py-2">
                            <p className="text-xs font-black uppercase tracking-wide text-emerald-700">
                              {bridgeCaptionLabel(caption.participant)}
                            </p>
                            <p className="mt-1 font-semibold text-slate-900">
                              {caption.translated_text}
                              {caption.target_language ? (
                                <span className="ml-2 text-xs font-bold uppercase text-emerald-700">
                                  {caption.passthrough ? "original" : caption.target_language}
                                </span>
                              ) : null}
                            </p>
                            {!caption.passthrough && caption.original_text !== caption.translated_text ? (
                              <p className="mt-1 text-xs text-slate-500">{caption.original_text}</p>
                            ) : null}
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                ) : null}
              </section>
            </div>
          ) : null}

          {activeSection === "summary" ? (
            <section className="space-y-6">
              <form
                onSubmit={handleSubmit}
                className="clinical-card grid gap-4 p-6 md:grid-cols-[minmax(0,2fr)_minmax(13rem,0.8fr)_auto]"
              >
                <Input
                  id="patientID"
                  label="Patient Name, Email, or ID"
                  icon={<Search className="h-5 w-5" />}
                  value={patientID}
                  onChange={(e) => setPatientID(e.target.value)}
                  placeholder="Jane Patient, jane@example.com, or UUID"
                  required
                />
                <div className="space-y-2">
                  <label
                    htmlFor="focus"
                    className="text-xs font-bold uppercase tracking-wide text-slate-600"
                  >
                    Summary Focus
                  </label>
                  <select
                    id="focus"
                    value={focus}
                    onChange={(e) => setFocus(e.target.value)}
                    className="h-12 w-full rounded-xl border border-slate-200 bg-white px-4 text-sm font-semibold text-slate-700 outline-none transition-all focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
                  >
                    {focusOptions.map((option) => (
                      <option key={option.value} value={option.value}>
                        {option.label}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex items-end">
                  <Button
                    type="submit"
                    variant="healthcare"
                    className="h-12 w-full"
                    disabled={isLoading}
                  >
                    <Sparkles className="h-4 w-4" />
                    {isLoading ? "Summarizing..." : "Generate"}
                  </Button>
                </div>
              </form>

              <div className="grid gap-6 xl:grid-cols-[1.25fr_0.75fr]">
                <section className="clinical-card min-h-[440px] p-6">
                  <h2 className="flex items-center gap-2 text-lg font-black">
                    <FileText className="h-5 w-5 text-indigo-600" />
                    Summary Output
                  </h2>
                  <div className="mt-6 rounded-2xl border border-slate-200 bg-slate-950 p-5 text-white">
                    {isLoading ? (
                      <div className="flex min-h-48 flex-col items-center justify-center">
                        <div className="mb-3 h-8 w-8 animate-spin rounded-full border-4 border-indigo-400 border-t-transparent" />
                        <p className="text-xs font-bold text-slate-300">
                          Generating clinical records translation...
                        </p>
                      </div>
                    ) : (
                      <pre className="min-h-48 whitespace-pre-wrap text-sm leading-7 text-slate-100">
                        {summary ||
                          "Input a patient name, email, or ID and focus option to generate a clinician-grounded summary file."}
                      </pre>
                    )}
                  </div>
                  <AuditTimeline events={auditEvents.slice(0, 4)} />
                </section>

                <IncidentPanel
                  incidents={incidents.slice(0, 4)}
                  loading={isRefreshingIncidents}
                  onRefresh={() => void loadIncidents(true)}
                />
              </div>
            </section>
          ) : null}

          {activeSection === "meetings" ? (
            <MeetingPanel
              meetings={meetings}
              loading={isRefreshingMeetings}
              mutatingMeetingID={mutatingMeetingID}
              onRefresh={() => void loadMeetings(true)}
              onAccept={(meetingID) => void handleAcceptMeeting(meetingID)}
              onCancel={(meetingID) => void handleCancelMeeting(meetingID)}
              onReschedule={(meetingID, payload) =>
                void handleRescheduleMeeting(meetingID, payload)
              }
              showScheduleForm={showScheduleForm}
              scheduleForm={scheduleForm}
              setScheduleForm={setScheduleForm}
              setShowScheduleForm={setShowScheduleForm}
              submittingSchedule={submittingSchedule}
              onSchedule={(e) => void handleScheduleMeeting(e)}
            />
          ) : null}

          {activeSection === "staff" ? (
            <StaffRegistrationPanel
              notice={staffNotice}
              staffRole={staffRole}
              onRegister={handleRegisterStaff}
            />
          ) : null}

          {activeSection === "consent" ? (
            <ConsentRequestPanel
              requests={consentRequests}
              qrCode={consentQRCode}
              loading={isRefreshingConsentRequests}
              onRefresh={() => void loadConsentRequests(true)}
              onCreate={handleCreateConsentRequest}
            />
          ) : null}

          {activeSection === "incidents" ? (
            <IncidentPanel
              incidents={incidents}
              loading={isRefreshingIncidents}
              onRefresh={() => void loadIncidents(true)}
              expanded
            />
          ) : null}

          {activeSection === "audit" ? (
            <AuditSearch
              patientID={patientID}
              setPatientID={setPatientID}
              setError={setError}
              events={auditEvents}
              setEvents={setAuditEvents}
            />
          ) : null}
      </DashboardShell>
  );
}

export default function HospitalPatientSummaryPage() {
  return (
    <Suspense
      fallback={
        <main className="min-h-screen bg-slate-100 px-5 py-8 dark:bg-slate-950">
          <div className="mx-auto max-w-7xl space-y-6">
            <div className="h-28 rounded-2xl bg-white shadow-sm dark:bg-slate-900" />
            <div className="grid gap-4 md:grid-cols-4">
              <div className="h-28 rounded-2xl bg-white shadow-sm dark:bg-slate-900" />
              <div className="h-28 rounded-2xl bg-white shadow-sm dark:bg-slate-900" />
              <div className="h-28 rounded-2xl bg-white shadow-sm dark:bg-slate-900" />
              <div className="h-28 rounded-2xl bg-white shadow-sm dark:bg-slate-900" />
            </div>
          </div>
        </main>
      }
    >
      <HospitalPatientSummaryPageContent />
    </Suspense>
  );
}

function PatientHomePanel({
  patients,
  loading,
  query,
  onQueryChange,
  onSearch,
  onRefresh,
  onSelectSummary,
  onSelectAudit,
}: {
  patients: HospitalPatientRecord[];
  loading: boolean;
  query: string;
  onQueryChange: (value: string) => void;
  onSearch: () => void;
  onRefresh: () => void;
  onSelectSummary: (patient: HospitalPatientRecord) => void;
  onSelectAudit: (patient: HospitalPatientRecord) => void;
}) {
  return (
    <section className="clinical-card p-6">
      <div className="flex flex-col gap-4 border-b border-slate-100 pb-5 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <h2 className="flex items-center gap-2 text-lg font-black">
            <Shield className="h-5 w-5 text-emerald-600" />
            Consented Patients
          </h2>
          <p className="mt-1 text-sm font-semibold text-slate-500">
            Patients with active hospital consent. Search by name, email, or patient ID.
          </p>
        </div>
        <form
          className="grid w-full gap-3 sm:grid-cols-[minmax(0,1fr)_auto_auto] lg:max-w-2xl"
          onSubmit={(event) => {
            event.preventDefault();
            onSearch();
          }}
        >
          <Input
            label="Patient Lookup"
            icon={<Search className="h-5 w-5" />}
            value={query}
            onChange={(event) => onQueryChange(event.target.value)}
            placeholder="Name, email, or UUID"
          />
          <div className="flex items-end">
            <Button type="submit" variant="healthcare" className="h-12 w-full" disabled={loading}>
              <Search className="h-4 w-4" />
              Search
            </Button>
          </div>
          <div className="flex items-end">
            <Button type="button" variant="outline" className="h-12 w-full" disabled={loading} onClick={onRefresh}>
              <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
              All
            </Button>
          </div>
        </form>
      </div>

      <div className="mt-5 grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {patients.map((patient) => (
          <PatientCard
            key={patient.patient_id}
            patient={patient}
            onSummary={() => onSelectSummary(patient)}
            onAudit={() => onSelectAudit(patient)}
          />
        ))}
      </div>

      {patients.length === 0 ? (
        <div className="mt-5 rounded-2xl border border-dashed border-slate-200 bg-slate-50 px-4 py-12 text-center text-sm font-semibold text-slate-400">
          {loading ? "Loading consented patients..." : "No active hospital consents matched."}
        </div>
      ) : null}
    </section>
  );
}

function PatientCard({
  patient,
  onSummary,
  onAudit,
}: {
  patient: HospitalPatientRecord;
  onSummary: () => void;
  onAudit: () => void;
}) {
  return (
    <article className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="truncate text-base font-black text-slate-950">
            {patient.full_name || "Unnamed patient"}
          </h3>
          <p className="mt-1 truncate text-sm font-semibold text-slate-500">
            {patient.email || "No email on file"}
          </p>
        </div>
        <span className="rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-black uppercase text-emerald-700">
          Active
        </span>
      </div>
      <div className="mt-4 space-y-2 text-xs font-semibold text-slate-500">
        <p className="break-all">ID: {patient.patient_id}</p>
        <p>Phone: {patient.phone_number || "Not listed"}</p>
        <p>Consent: {formatDateTime(patient.consent_granted_at, "Granted")}</p>
        <p>Last call: {formatDateTime(patient.last_call_at, "No calls yet")}</p>
      </div>
      <div className="mt-5 flex flex-wrap gap-2">
        <Button type="button" size="sm" variant="healthcare" onClick={onSummary}>
          <Sparkles className="h-4 w-4" />
          Summary
        </Button>
        <Button type="button" size="sm" variant="outline" onClick={onAudit}>
          <ListChecks className="h-4 w-4" />
          Audit
        </Button>
      </div>
    </article>
  );
}

function MeetingPanel({
  meetings,
  loading,
  mutatingMeetingID,
  onRefresh,
  onAccept,
  onCancel,
  onReschedule,
  showScheduleForm,
  scheduleForm,
  setScheduleForm,
  setShowScheduleForm,
  submittingSchedule,
  onSchedule,
}: {
  meetings: HospitalMeetingRecord[];
  loading: boolean;
  mutatingMeetingID: string | null;
  onRefresh: () => void;
  onAccept: (meetingID: string) => void;
  onCancel: (meetingID: string) => void;
  onReschedule: (
    meetingID: string,
    payload: HTTPHospitalMeetingRescheduleRequest,
  ) => void;
  showScheduleForm?: boolean;
  scheduleForm?: {
    patient_id: string;
    starts_at: string;
    duration_minutes: number;
    timezone: string;
    title: string;
    notes: string;
  };
  setScheduleForm?: React.Dispatch<React.SetStateAction<{
    patient_id: string;
    starts_at: string;
    duration_minutes: number;
    timezone: string;
    title: string;
    notes: string;
  }>>;
  setShowScheduleForm?: (v: boolean) => void;
  submittingSchedule?: boolean;
  onSchedule?: (e: React.FormEvent) => void;
}) {
  return (
    <section className="clinical-card p-6">
      <div className="flex flex-col gap-3 border-b border-slate-100 pb-5 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="flex items-center gap-2 text-lg font-black">
            <CalendarDays className="h-5 w-5 text-orange-500" />
            Visit Requests
          </h2>
          <p className="mt-1 text-sm font-semibold text-slate-500">
            Accept pending patient requests, schedule new visits, or reschedule.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            size="sm"
            variant="healthcare"
            onClick={() => setShowScheduleForm?.(!showScheduleForm)}
          >
            <Video className="h-4 w-4" />
            {showScheduleForm ? "Back to List" : "Schedule New"}
          </Button>
          <button
            type="button"
            disabled={loading}
            onClick={onRefresh}
            className="inline-flex h-10 w-10 items-center justify-center rounded-xl text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-800 disabled:opacity-50"
          >
            <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
          </button>
        </div>
      </div>

      {showScheduleForm && onSchedule && scheduleForm && setScheduleForm ? (
        <form onSubmit={onSchedule} className="mt-5 grid gap-4 rounded-2xl border border-slate-200 bg-slate-50/70 p-5 md:grid-cols-2">
          <div className="space-y-2">
            <label className="text-xs font-black uppercase tracking-wide text-slate-500">Patient ID</label>
            <input
              value={scheduleForm.patient_id}
              onChange={(e) => setScheduleForm((prev) => ({ ...prev, patient_id: e.target.value }))}
              placeholder="Enter patient ID"
              className="h-10 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
              required
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-black uppercase tracking-wide text-slate-500">Date & Time</label>
            <input
              type="datetime-local"
              value={scheduleForm.starts_at}
              onChange={(e) => setScheduleForm((prev) => ({ ...prev, starts_at: e.target.value }))}
              className="h-10 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
              required
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-black uppercase tracking-wide text-slate-500">Duration (min)</label>
            <input
              type="number"
              min={15}
              step={15}
              value={scheduleForm.duration_minutes}
              onChange={(e) => setScheduleForm((prev) => ({ ...prev, duration_minutes: Number(e.target.value) || 30 }))}
              className="h-10 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
              required
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-black uppercase tracking-wide text-slate-500">Timezone (IANA)</label>
            <input
              value={scheduleForm.timezone}
              onChange={(e) => setScheduleForm((prev) => ({ ...prev, timezone: e.target.value }))}
              placeholder="America/Chicago"
              className="h-10 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
              required
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-black uppercase tracking-wide text-slate-500">Title</label>
            <input
              value={scheduleForm.title}
              onChange={(e) => setScheduleForm((prev) => ({ ...prev, title: e.target.value }))}
              placeholder="Zorba Health video visit"
              className="h-10 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
            />
          </div>
          <div className="space-y-2 md:col-span-2">
            <label className="text-xs font-black uppercase tracking-wide text-slate-500">Notes (optional)</label>
            <textarea
              value={scheduleForm.notes}
              onChange={(e) => setScheduleForm((prev) => ({ ...prev, notes: e.target.value }))}
              rows={2}
              className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
            />
          </div>
          <div className="md:col-span-2 flex justify-end">
            <Button type="submit" variant="healthcare" disabled={submittingSchedule} className="h-10">
              {submittingSchedule ? "Scheduling..." : "Schedule Meeting"}
            </Button>
          </div>
        </form>
      ) : null}

      {!showScheduleForm ? (
        <div className="mt-5 grid gap-4">
          {meetings.map((meeting) => (
            <MeetingCard
              key={meeting.id}
              meeting={meeting}
              mutating={mutatingMeetingID === meeting.id}
              onAccept={() => onAccept(meeting.id)}
              onCancel={() => onCancel(meeting.id)}
              onReschedule={(payload) => onReschedule(meeting.id, payload)}
            />
          ))}

          {meetings.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-slate-200 bg-slate-50 px-4 py-12 text-center text-sm font-semibold text-slate-400">
              No patient visit requests are waiting for this hospital.
            </div>
          ) : null}
        </div>
      ) : null}
    </section>
  );
}

function StaffRegistrationPanel({
  notice,
  staffRole,
  onRegister,
}: {
  notice: string;
  staffRole?: string;
  onRegister: (payload: HTTPHospitalStaffRegisterRequest) => Promise<boolean>;
}) {
  const [form, setForm] = useState({
    staffName: "",
    email: "",
    phoneNumber: "",
    password: "",
    staffRole: "doctor",
  });
  const [submitting, setSubmitting] = useState(false);

  const canRegister = staffRole === "admin";

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitting(true);
    const ok = await onRegister({
      staff_name: form.staffName.trim(),
      email: form.email.trim(),
      phone_number: form.phoneNumber.trim() || undefined,
      password: form.password,
      staff_role: form.staffRole,
    });
    if (ok) {
      setForm({
        staffName: "",
        email: "",
        phoneNumber: "",
        password: "",
        staffRole: "doctor",
      });
    }
    setSubmitting(false);
  };

  return (
    <section className="clinical-card p-6">
      <div className="border-b border-slate-100 pb-5">
        <h2 className="flex items-center gap-2 text-lg font-black">
          <UserPlus className="h-5 w-5 text-indigo-600" />
          Register Hospital Staff
        </h2>
        <p className="mt-1 text-sm font-semibold text-slate-500">
          Create staff logins linked to this hospital. New staff can sign in from the hospital portal.
        </p>
      </div>

      {notice ? (
        <div className="mt-5 rounded-xl border border-emerald-200 bg-emerald-50/80 p-4 text-sm font-semibold text-emerald-700">
          {notice}
        </div>
      ) : null}

      {!canRegister ? (
        <div className="mt-5 rounded-xl border border-orange-200 bg-orange-50/80 p-4 text-sm font-semibold text-orange-700">
          Only hospital admin staff can register additional staff accounts.
        </div>
      ) : null}

      <form onSubmit={handleSubmit} className="mt-5 grid gap-4 md:grid-cols-2">
        <Input
          label="Staff Name"
          icon={<UserPlus className="h-5 w-5" />}
          value={form.staffName}
          onChange={(event) => setForm({ ...form, staffName: event.target.value })}
          placeholder="Dr. Jane Clinician"
          disabled={!canRegister}
          required
        />
        <Input
          label="Staff Email"
          icon={<Search className="h-5 w-5" />}
          type="email"
          value={form.email}
          onChange={(event) => setForm({ ...form, email: event.target.value })}
          placeholder="clinician@hospital.org"
          disabled={!canRegister}
          required
        />
        <Input
          label="Phone Number"
          icon={<Clock className="h-5 w-5" />}
          value={form.phoneNumber}
          onChange={(event) => setForm({ ...form, phoneNumber: event.target.value })}
          placeholder="+15551234567"
          disabled={!canRegister}
        />
        <Input
          label="Temporary Password"
          icon={<Shield className="h-5 w-5" />}
          type="password"
          minLength={8}
          value={form.password}
          onChange={(event) => setForm({ ...form, password: event.target.value })}
          placeholder="Create initial password"
          disabled={!canRegister}
          required
        />
        <div className="space-y-2">
          <label
            htmlFor="staff-role"
            className="text-xs font-bold uppercase tracking-wide text-slate-600"
          >
            Staff Role
          </label>
          <select
            id="staff-role"
            value={form.staffRole}
            onChange={(event) => setForm({ ...form, staffRole: event.target.value })}
            disabled={!canRegister}
            className="h-12 w-full rounded-xl border border-slate-200 bg-white px-4 text-sm font-semibold text-slate-700 outline-none transition-all focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100 disabled:bg-slate-100 disabled:text-slate-400"
          >
            <option value="doctor">Doctor</option>
            <option value="nurse">Nurse</option>
            <option value="billing">Billing</option>
            <option value="admin">Admin</option>
          </select>
        </div>
        <div className="flex items-end">
          <Button
            type="submit"
            variant="healthcare"
            className="h-12 w-full"
            disabled={!canRegister || submitting}
          >
            <UserPlus className="h-4 w-4" />
            {submitting ? "Registering..." : "Register Staff"}
          </Button>
        </div>
      </form>
    </section>
  );
}

const hospitalConsentPermissionOptions = [
  { value: "HEALTH_RECORD_ACCESS", label: "Health records" },
  { value: "AI_SUMMARIZATION", label: "AI summaries" },
  { value: "SCHEDULING", label: "Scheduling" },
  { value: "EMERGENCY_ESCALATION", label: "Emergency escalation" },
];

function ConsentRequestPanel({
  requests,
  qrCode,
  loading,
  onRefresh,
  onCreate,
}: {
  requests: HospitalConsentRequestRecord[];
  qrCode: string;
  loading: boolean;
  onRefresh: () => void;
  onCreate: (payload: HTTPHospitalConsentRequestCreateRequest) => Promise<boolean>;
}) {
  const [note, setNote] = useState("");
  const [expiresIn, setExpiresIn] = useState("30");
  const [permissions, setPermissions] = useState([
    "HEALTH_RECORD_ACCESS",
    "AI_SUMMARIZATION",
    "SCHEDULING",
  ]);
  const [submitting, setSubmitting] = useState(false);

  const togglePermission = (value: string) => {
    setPermissions((current) =>
      current.includes(value)
        ? current.filter((item) => item !== value)
        : [...current, value],
    );
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setSubmitting(true);
    const ok = await onCreate({
      requested_permissions: permissions,
      note: note.trim() || undefined,
      expires_in_minutes: Number(expiresIn) || 30,
    });
    if (ok) {
      setNote("");
    }
    setSubmitting(false);
  };

  const latest = requests[0];

  return (
    <section className="clinical-card p-6">
      <div className="flex flex-col gap-3 border-b border-slate-100 pb-5 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="flex items-center gap-2 text-lg font-black">
            <QrCode className="h-5 w-5 text-indigo-600" />
            Patient Consent QR
          </h2>
          <p className="mt-1 text-sm font-semibold text-slate-500">
            Generate a patient-scannable request that grants this hospital access after patient confirmation.
          </p>
        </div>
        <button
          type="button"
          disabled={loading}
          onClick={onRefresh}
          className="inline-flex h-10 w-10 items-center justify-center rounded-xl text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-800 disabled:opacity-50"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
        </button>
      </div>

      <div className="mt-5 grid gap-6 xl:grid-cols-[0.95fr_1.05fr]">
        <form onSubmit={handleSubmit} className="space-y-4">
          <Input
            label="Request note"
            icon={<FileText className="h-5 w-5" />}
            value={note}
            onChange={(event) => setNote(event.target.value)}
            placeholder="Reason for access"
          />
          <div className="space-y-2">
            <label className="text-xs font-bold uppercase tracking-wide text-slate-600">
              Expires In
            </label>
            <select
              value={expiresIn}
              onChange={(event) => setExpiresIn(event.target.value)}
              className="h-12 w-full rounded-xl border border-slate-200 bg-white px-4 text-sm font-semibold text-slate-700 outline-none transition-all focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
            >
              <option value="15">15 minutes</option>
              <option value="30">30 minutes</option>
              <option value="60">1 hour</option>
              <option value="240">4 hours</option>
              <option value="1440">24 hours</option>
            </select>
          </div>
          <div className="grid gap-2 sm:grid-cols-2">
            {hospitalConsentPermissionOptions.map((option) => {
              const selected = permissions.includes(option.value);
              return (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => togglePermission(option.value)}
                  className={`rounded-2xl border px-4 py-3 text-left text-sm font-black transition-colors ${
                    selected
                      ? "border-indigo-200 bg-indigo-50 text-indigo-700"
                      : "border-slate-200 bg-white text-slate-500 hover:bg-slate-50"
                  }`}
                >
                  {option.label}
                </button>
              );
            })}
          </div>
          <Button
            type="submit"
            variant="healthcare"
            className="h-12 w-full"
            disabled={submitting || permissions.length === 0}
          >
            <QrCode className="h-4 w-4" />
            {submitting ? "Generating..." : "Generate QR"}
          </Button>
        </form>

        <div className="rounded-2xl border border-slate-200 bg-slate-50/80 p-5">
          <div className="grid gap-5 md:grid-cols-[13rem_1fr]">
            <div className="flex aspect-square items-center justify-center rounded-2xl border border-slate-200 bg-white p-4">
              {qrCode ? (
                <Image
                  src={qrCode}
                  alt="Hospital consent request QR code"
                  width={196}
                  height={196}
                  className="h-full w-full"
                  unoptimized
                />
              ) : (
                <QrCode className="h-16 w-16 text-slate-300" />
              )}
            </div>
            <div className="min-w-0">
              <p className="text-xs font-black uppercase tracking-wide text-slate-500">
                Latest request
              </p>
              <h3 className="mt-2 text-lg font-black text-slate-950">
                {latest?.hospital_name || "No QR generated yet"}
              </h3>
              <p className="mt-2 break-all text-xs font-semibold leading-5 text-slate-500">
                {latest?.token || "Create a request to show a scannable code."}
              </p>
              {latest ? (
                <div className="mt-4 flex flex-wrap gap-2">
                  <StatusPill status={latest.status} />
                  <span className="rounded-full bg-white px-3 py-1 text-xs font-black text-slate-500">
                    Expires {formatDateTime(latest.expires_at)}
                  </span>
                </div>
              ) : null}
            </div>
          </div>
        </div>
      </div>

      <div className="mt-6 grid gap-3">
        {requests.map((request) => (
          <div
            key={request.id || request.token}
            className="rounded-2xl border border-slate-200 bg-white p-4"
          >
            <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <StatusPill status={request.status} />
                  <span className="text-xs font-semibold text-slate-400">
                    {formatDateTime(request.created_at, "Created recently")}
                  </span>
                </div>
                <p className="mt-3 text-sm font-black text-slate-950">
                  Patient: {request.patient_id || "Claimed on scan"}
                </p>
                <p className="mt-1 break-all text-xs font-semibold text-slate-500">
                  {request.token}
                </p>
              </div>
              <div className="flex flex-wrap gap-2 sm:justify-end">
                {(request.requested_permissions ?? []).map((permission) => (
                  <span
                    key={permission}
                    className="rounded-full bg-slate-100 px-3 py-1 text-xs font-black text-slate-600"
                  >
                    {permission.replaceAll("_", " ")}
                  </span>
                ))}
              </div>
            </div>
          </div>
        ))}
        {requests.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-slate-200 bg-slate-50 px-4 py-10 text-center text-sm font-semibold text-slate-400">
            No consent requests have been generated yet.
          </div>
        ) : null}
      </div>
    </section>
  );
}

function StatusPill({ status }: { status?: string }) {
  const normalized = status || "pending";
  const color =
    normalized === "approved"
      ? "bg-emerald-50 text-emerald-700"
      : normalized === "expired" || normalized === "revoked"
        ? "bg-slate-100 text-slate-500"
        : "bg-orange-50 text-orange-700";
  return (
    <span className={`rounded-full px-3 py-1 text-xs font-black uppercase ${color}`}>
      {normalized}
    </span>
  );
}

function MeetingCard({
  meeting,
  mutating,
  onAccept,
  onCancel,
  onReschedule,
}: {
  meeting: HospitalMeetingRecord;
  mutating: boolean;
  onAccept: () => void;
  onCancel: () => void;
  onReschedule: (payload: HTTPHospitalMeetingRescheduleRequest) => void;
}) {
  const [startsAt, setStartsAt] = useState(toDateTimeLocalValue(meeting.starts_at));
  const [duration, setDuration] = useState(String(meeting.duration_minutes || 30));
  const [timezone, setTimezone] = useState(() =>
    resolveIANATimezone(meeting.timezone),
  );
  const [title, setTitle] = useState(meeting.title || "Zorba Health video visit");

  useEffect(() => {
    setStartsAt(toDateTimeLocalValue(meeting.starts_at));
    setDuration(String(meeting.duration_minutes || 30));
    setTimezone(resolveIANATimezone(meeting.timezone));
    setTitle(meeting.title || "Zorba Health video visit");
  }, [meeting.duration_minutes, meeting.starts_at, meeting.timezone, meeting.title]);

  const status = meeting.status || "pending";
  const pending = status === "pending";
  const cancelled = status === "cancelled";

  const handleReschedule = (event: React.FormEvent) => {
    event.preventDefault();
    if (!startsAt) return;
    onReschedule({
      starts_at: dateTimeLocalToRFC3339(startsAt),
      duration_minutes: Number(duration) || 30,
      timezone: resolveIANATimezone(timezone),
      title,
    });
  };

  return (
    <article className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span
              className={`inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-black uppercase ${
                pending
                  ? "bg-orange-50 text-orange-700"
                  : cancelled
                    ? "bg-slate-100 text-slate-500"
                    : "bg-emerald-50 text-emerald-700"
              }`}
            >
              {pending ? <Clock className="h-3.5 w-3.5" /> : <CheckCircle className="h-3.5 w-3.5" />}
              {formatMeetingStatus(status)}
            </span>
            <span className="text-xs font-semibold text-slate-400">
              {meeting.duration_minutes || 30} min
            </span>
          </div>
          <h3 className="mt-3 text-base font-black text-slate-950">
            {meeting.title || "Zorba Health video visit"}
          </h3>
          <div className="mt-3 grid gap-2 text-sm font-semibold text-slate-500 sm:grid-cols-2">
            <p>Patient ID: {meeting.patient_id || "Not attached"}</p>
            <p>{formatDateTime(meeting.starts_at, "Time not set")}</p>
            <p>Staff ID: {meeting.staff_id || "Unassigned"}</p>
            <p>{meeting.timezone || "UTC"}</p>
          </div>
        </div>

        <div className="flex flex-wrap gap-2 lg:justify-end">
          {pending ? (
            <Button
              type="button"
              size="sm"
              variant="healthcare"
              disabled={mutating}
              onClick={onAccept}
            >
              <Video className="h-4 w-4" />
              Accept
            </Button>
          ) : null}
          {meeting.join_url ? (
            <Button type="button" size="sm" variant="outline" asChild>
              <a href={meeting.join_url} target="_blank" rel="noreferrer">
                <ExternalLink className="h-4 w-4" />
                Join
              </a>
            </Button>
          ) : null}
          {!cancelled ? (
            <Button
              type="button"
              size="sm"
              variant="secondary"
              disabled={mutating}
              onClick={onCancel}
            >
              <XCircle className="h-4 w-4" />
              Cancel
            </Button>
          ) : null}
        </div>
      </div>

      {!cancelled ? (
        <form
          onSubmit={handleReschedule}
          className="mt-5 grid gap-3 rounded-2xl border border-slate-100 bg-slate-50/70 p-4 md:grid-cols-[1fr_8rem_12rem_1fr_auto]"
        >
          <div className="space-y-2">
            <label className="text-xs font-black uppercase tracking-wide text-slate-500">
              New Time
            </label>
            <input
              type="datetime-local"
              value={startsAt}
              onChange={(event) => setStartsAt(event.target.value)}
              className="h-10 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
              required
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-black uppercase tracking-wide text-slate-500">
              Minutes
            </label>
            <input
              type="number"
              min={15}
              step={15}
              value={duration}
              onChange={(event) => setDuration(event.target.value)}
              className="h-10 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
              required
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-black uppercase tracking-wide text-slate-500">
              Timezone (IANA)
            </label>
            <input
              value={timezone}
              onChange={(event) => setTimezone(event.target.value)}
              placeholder="America/Chicago"
              className="h-10 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
              required
            />
          </div>
          <div className="space-y-2">
            <label className="text-xs font-black uppercase tracking-wide text-slate-500">
              Title
            </label>
            <input
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              className="h-10 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
            />
          </div>
          <div className="flex items-end">
            <Button type="submit" size="sm" variant="outline" disabled={mutating} className="h-10 w-full">
              {mutating ? "Saving..." : "Reschedule"}
            </Button>
          </div>
        </form>
      ) : null}
    </article>
  );
}

function IncidentPanel({
  incidents,
  loading,
  onRefresh,
  expanded = false,
}: {
  incidents: HospitalIncidentRecord[];
  loading: boolean;
  onRefresh: () => void;
  expanded?: boolean;
}) {
  return (
    <aside className={`clinical-card p-6 ${expanded ? "" : "xl:sticky xl:top-28"}`}>
      <div className="flex items-center justify-between gap-3">
        <h2 className="flex items-center gap-2 text-lg font-black">
          <AlertCircle className="h-5 w-5 text-rose-600" />
          Emergency Incidents
        </h2>
        <button
          type="button"
          disabled={loading}
          onClick={onRefresh}
          className="rounded-lg p-2 text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-800"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? "animate-spin" : ""}`} />
        </button>
      </div>
      <div className="mt-5 space-y-4">
        {incidents.map((incident) => {
          const high =
            incident.severity?.toLowerCase() === "critical" ||
            incident.severity?.toLowerCase() === "high";
          return (
            <div
              key={incident.event_id}
              className={`rounded-2xl border p-4 ${
                high ? "border-rose-200 bg-rose-50/70" : "border-orange-200 bg-orange-50/60"
              }`}
            >
              <div className="flex items-center justify-between gap-3">
                <span
                  className={`inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-black uppercase ${
                    high ? "bg-rose-100 text-rose-700" : "bg-orange-100 text-orange-700"
                  }`}
                >
                  {high ? <span className="h-2 w-2 rounded-full bg-rose-500" /> : null}
                  {incident.severity || "Emergency"}
                </span>
                <span className="text-xs font-semibold text-slate-500">
                  {formatTimeOnly(incident.timestamp)}
                </span>
              </div>
              <p className="mt-3 text-sm font-black text-slate-900">
                Patient ID: {incident.patient_id || "Not attached"}
              </p>
              <p className="mt-1 text-xs font-semibold text-slate-500">
                {incident.service_name || "Voice Triage"}
              </p>
              {incident.failure_reason ? (
                <p className="mt-3 rounded-xl bg-white p-3 text-xs font-semibold leading-5 text-rose-700">
                  {incident.failure_reason}
                </p>
              ) : null}
            </div>
          );
        })}
        {incidents.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-slate-200 bg-slate-50 px-4 py-10 text-center text-sm font-semibold text-slate-400">
            No active emergency alerts in queue.
          </div>
        ) : null}
      </div>
    </aside>
  );
}

function AuditTimeline({ events }: { events: AuditEventRecord[] }) {
  return (
    <div className="mt-8 border-t border-slate-100 pt-6">
      <h3 className="flex items-center gap-2 text-sm font-black">
        <Activity className="h-4 w-4 text-indigo-600" />
        Audit Trail Trace
      </h3>
      <div className="mt-4 space-y-3">
        {events.length > 0 ? (
          events.map((event) => (
            <div
              key={event.event_id}
              className="rounded-2xl border border-slate-200 bg-slate-50/70 p-4 text-xs"
            >
              <div className="flex items-start justify-between gap-4">
                <div>
                  <p className="font-black uppercase tracking-wide text-slate-800">
                    {formatEventType(event.event_type)}
                  </p>
                  <p className="mt-1 text-slate-400">
                    Service: {event.service_name || "Record Manager"}
                  </p>
                </div>
                <p className="flex items-center gap-1 text-slate-400">
                  <Clock className="h-3.5 w-3.5" />
                  {event.timestamp
                    ? formatTimeOnly(event.timestamp, "Unknown")
                    : "Unknown"}
                </p>
              </div>
            </div>
          ))
        ) : (
          <p className="text-xs font-semibold text-slate-400">
            Query a patient record to retrieve compliant access audit events.
          </p>
        )}
      </div>
    </div>
  );
}

function AuditSearch({
  patientID,
  setPatientID,
  setError,
  events,
  setEvents,
}: {
  patientID: string;
  setPatientID: (value: string) => void;
  setError: (value: string) => void;
  events: AuditEventRecord[];
  setEvents: (events: AuditEventRecord[]) => void;
}) {
  const [loading, setLoading] = useState(false);
  const [typeFilter, setTypeFilter] = useState("all");
  const [startFilter, setStartFilter] = useState("");
  const [endFilter, setEndFilter] = useState("");

  const eventTypes = useMemo(() => getAuditEventTypeOptions(events), [events]);
  const filteredEvents = useMemo(
    () => filterAuditEvents(events, typeFilter, startFilter, endFilter),
    [endFilter, events, startFilter, typeFilter],
  );

  const loadAudit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!patientID.trim()) return;
    setLoading(true);
    setError("");
    try {
      const response = await apiFetch(
        "hospital",
        `${APIEndpoints.HOSPITAL_PATIENT_AUDIT}?patient_id=${encodeURIComponent(patientID.trim())}`,
      );
      const data: HTTPHospitalPatientAuditResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to load audit trail.");
        return;
      }
      setEvents(data.data?.events ?? []);
    } catch {
      setError("Network error while loading audit trail.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <section className="clinical-card overflow-hidden">
      <form onSubmit={loadAudit} className="border-b border-slate-200 p-6">
        <div className="grid gap-4 md:grid-cols-[1fr_auto]">
          <Input
            label="Patient Name, Email, or ID"
            icon={<Search className="h-5 w-5" />}
            value={patientID}
            onChange={(e) => setPatientID(e.target.value)}
            placeholder="Jane Patient, jane@example.com, or UUID"
            required
          />
          <div className="flex items-end">
            <Button type="submit" variant="healthcare" className="h-12 w-full" disabled={loading}>
              <ListChecks className="h-4 w-4" />
              {loading ? "Searching..." : "Load Audit Trail"}
            </Button>
          </div>
        </div>
        <div className="mt-5 grid gap-3 md:grid-cols-3">
          <div className="space-y-2">
            <label htmlFor="hospital-audit-type" className="text-xs font-black uppercase tracking-wide text-slate-500">
              Event Type
            </label>
            <select
              id="hospital-audit-type"
              value={typeFilter}
              onChange={(event) => setTypeFilter(event.target.value)}
              className="h-11 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
            >
              <option value="all">All audit types</option>
              {eventTypes.map((eventType) => (
                <option key={eventType} value={eventType}>
                  {formatEventType(eventType)}
                </option>
              ))}
            </select>
          </div>
          <div className="space-y-2">
            <label htmlFor="hospital-audit-start" className="text-xs font-black uppercase tracking-wide text-slate-500">
              From
            </label>
            <input
              id="hospital-audit-start"
              type="datetime-local"
              value={startFilter}
              onChange={(event) => setStartFilter(event.target.value)}
              className="h-11 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
            />
          </div>
          <div className="space-y-2">
            <label htmlFor="hospital-audit-end" className="text-xs font-black uppercase tracking-wide text-slate-500">
              To
            </label>
            <input
              id="hospital-audit-end"
              type="datetime-local"
              value={endFilter}
              onChange={(event) => setEndFilter(event.target.value)}
              className="h-11 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
            />
          </div>
        </div>
      </form>
      <div className="overflow-x-auto">
        <table className="w-full min-w-[720px] text-left text-sm">
          <thead className="bg-slate-50 text-xs font-black uppercase tracking-wide text-slate-500">
            <tr>
              <th className="px-5 py-3">Event</th>
              <th className="px-5 py-3">Service</th>
              <th className="px-5 py-3">Status</th>
              <th className="px-5 py-3">Timestamp</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {filteredEvents.map((event, index) => (
              <tr key={event.event_id || index}>
                <td className="px-5 py-4 font-bold text-slate-900">
                  {formatEventType(event.event_type)}
                </td>
                <td className="px-5 py-4 text-slate-600">
                  {event.service_name || "Record Manager"}
                </td>
                <td className="px-5 py-4">
                  <span
                    className={`rounded-full px-2.5 py-1 text-xs font-black ${
                      event.success_status === false
                        ? "bg-rose-50 text-rose-700"
                        : "bg-emerald-50 text-emerald-700"
                    }`}
                  >
                    {event.success_status === false ? "Failed" : "Verified"}
                  </span>
                </td>
                <td className="px-5 py-4 text-slate-500">
                  {formatDateTime(event.timestamp)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {filteredEvents.length === 0 ? (
          <p className="p-6 text-sm font-semibold text-slate-400">
            No audit trail matches the current filters.
          </p>
        ) : null}
      </div>
    </section>
  );
}
