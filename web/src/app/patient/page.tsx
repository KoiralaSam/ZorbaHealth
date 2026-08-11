"use client";

export const dynamic = "force-dynamic";

import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { Button } from "../../components/ui/button";
import { type SidebarItem } from "../../components/ui/sidebar";
import { StatCard } from "../../components/ui/stat-card";
import { DashboardShell } from "../../components/layout/dashboard-shell";
import { PageHeader } from "../../components/layout/page-header";
import { StatusBanner } from "../../components/status-banner";
import { toast } from "sonner";
import {
  APIEndpoints,
  AuditEventRecord,
  BridgedCallSessionRecord,
  ConsentRecord,
  getAuditEventTypeOptions,
  HTTPBridgedCallSessionResponse,
  HTTPPatientAuditResponse,
  HTTPPatientCallListResponse,
  HTTPPatientConsentListResponse,
  HTTPPatientConsentMutationRequest,
  HTTPPatientConsentMutationResponse,
  HTTPPatientConsentRequestApproveResponse,
  HTTPPatientConsentRequestLookupResponse,
  HTTPPatientHealthAnswerRequest,
  HTTPPatientHealthAnswerResponse,
  HTTPPatientHospitalConsentListResponse,
  HospitalConsentRequestRecord,
  PatientHospitalConsentRecord,
  HTTPPatientProfileResponse,
  HTTPPatientWelfareCheckCreateRequest,
  HTTPPatientWelfareCheckListResponse,
  HTTPPatientWelfareCheckResponse,
  PatientCallSummary,
  HospitalMeetingRecord,
  HTTPPatientMeetingListResponse,
  HTTPPatientMeetingMutationResponse,
  HTTPPatientMeetingScheduleRequest,
  HTTPPatientSchedulableStaffResponse,
  RequestBridgedCallTransferRequest,
  UpdateBridgedCallTranslationRequest,
  InterpretationSegmentMessage,
  PatientSchedulableStaffRecord,
  PatientWelfareCheck,
  WelfareCheckReason,
} from "../../contracts";
import { apiFetch, cachedApiJSON, clearApiCache, logoutAuth, preloadApiJSON } from "../../lib/auth-client";
import { PatientAppointmentBookingPanel } from "../../components/PatientAppointmentBookingPanel";
import {
  findActivePatientCall,
  isCallInProgress,
  resolveBridgedCallSessionId,
} from "../../lib/bridged-call";
import { useAuth } from "../../hooks/useAuth";
import { usePatientLocationSession } from "../../hooks/usePatientLocationSession";
import {
  Calendar,
  CalendarDays,
  Camera,
  CheckCircle,
  Clock,
  Compass,
  FileText,
  LayoutDashboard,
  Mail,
  MessageSquare,
  Phone,
  QrCode,
  RefreshCw,
  Send,
  Shield,
  ToggleLeft,
  Trash2,
  User,
  Languages,
  Video,
  VideoOff,
  HeartPulse,
} from "lucide-react";
import { Room, RoomEvent } from "livekit-client";
import { formatDateTime, formatEventType, formatTimeOnly, meaningfulDate } from "../../lib/format";
import { resolveIANATimezone } from "../../lib/timezone";

const consentLabels: Record<string, string> = {
  VOICE_ASSISTANT_USE: "Voice assistant use",
  HEALTH_RECORD_ACCESS: "Health record access",
  LOCATION_ACCESS: "Location sharing during emergencies",
  SMS_NOTIFICATION: "SMS notifications",
  EMAIL_NOTIFICATION: "Email notifications",
  AI_SUMMARIZATION: "AI summarization",
  THIRD_PARTY_MODEL_PROCESSING: "Third-party model processing",
};

const consentTypes = Object.keys(consentLabels);

const welfareReasonLabels: Record<WelfareCheckReason, string> = {
  medication_reminder: "Medication reminder",
  mental_wellbeing: "Mental wellbeing",
  daily_checkup: "Daily checkup",
  symptom_follow_up: "Symptom follow-up",
  care_plan_reminder: "Care plan reminder",
  other: "Other",
};

const welfareReasons = Object.keys(welfareReasonLabels) as WelfareCheckReason[];

const localDateTimeValue = () => {
  const date = new Date(Date.now() + 60 * 60 * 1000);
  date.setMinutes(Math.ceil(date.getMinutes() / 5) * 5, 0, 0);
  return new Date(date.getTime() - date.getTimezoneOffset() * 60_000)
    .toISOString()
    .slice(0, 16);
};

const navItems: SidebarItem[] = [
  { id: "dashboard", label: "Dashboard", icon: LayoutDashboard },
  { id: "consents", label: "Consents", icon: ToggleLeft },
  { id: "records", label: "Ask Records", icon: MessageSquare },
  { id: "calls", label: "Call History", icon: Phone },
  { id: "meetings", label: "Meetings", icon: Video },
  { id: "appointments", label: "Appointments", icon: CalendarDays },
  { id: "welfare", label: "Welfare Checks", icon: HeartPulse },
  { id: "audit", label: "Audit Trail", icon: Shield },
  { id: "gps", label: "GPS", icon: Compass },
];

function bridgeCaptionLabel(participant?: string) {
  return participant === "staff" ? "Clinician" : "Patient";
}

function normalizePatientCalls(payload: HTTPPatientCallListResponse) {
  const data = payload.data as
    | (HTTPPatientCallListResponse["data"] & {
        data?: { calls?: PatientCallSummary[] };
        items?: PatientCallSummary[];
      })
    | undefined;
  const calls = data?.calls ?? data?.data?.calls ?? data?.items ?? [];
  return [...calls].sort((a, b) => {
    const aTime = meaningfulDate(a.started_at)?.getTime() ?? 0;
    const bTime = meaningfulDate(b.started_at)?.getTime() ?? 0;
    return bTime - aTime;
  });
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

function PatientHomePageContent() {
  const router = useRouter();
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const activeSection = searchParams.get("section") || "dashboard";
  const setActiveSection = useCallback(
    (nextSection: string) => {
      const params = new URLSearchParams(searchParams.toString());
      params.set("section", nextSection);
      router.replace(`${pathname}?${params.toString()}`, { scroll: false });
    },
    [pathname, router, searchParams],
  );
  const { ready, authenticated, accessToken } = useAuth("patient");
  const sectionNavItems = useMemo(
    () =>
      navItems.map((item) => ({
        ...item,
        href: `${pathname}?section=${item.id}`,
      })),
    [pathname],
  );

  useEffect(() => {
    if (ready && !authenticated) {
      router.replace("/login/patient");
    }
  }, [ready, authenticated, router]);

  const [profile, setProfile] =
    useState<HTTPPatientProfileResponse["data"] | null>(null);
  const [consents, setConsents] = useState<ConsentRecord[]>([]);
  const [hospitalConsents, setHospitalConsents] = useState<PatientHospitalConsentRecord[]>([]);
  const [calls, setCalls] = useState<PatientCallSummary[]>([]);
  const [welfareChecks, setWelfareChecks] = useState<PatientWelfareCheck[]>([]);
  const [auditEvents, setAuditEvents] = useState<AuditEventRecord[]>([]);
  const [question, setQuestion] = useState("");
  const [answer, setAnswer] = useState("");
  const [answerSources, setAnswerSources] = useState<string[]>([]);
  const [recordsNotice, setRecordsNotice] = useState("");
  const [error, setError] = useState("");
  const [locationNotice, setLocationNotice] = useState("");
  const [loading, setLoading] = useState(true);
  const [mutatingConsent, setMutatingConsent] = useState<string | null>(null);
  const [askingQuestion, setAskingQuestion] = useState(false);
  const [welfareScheduledAt, setWelfareScheduledAt] = useState(localDateTimeValue);
  const [welfareReason, setWelfareReason] =
    useState<WelfareCheckReason>("daily_checkup");
  const [welfareDetail, setWelfareDetail] = useState("");
  const [creatingWelfareCheck, setCreatingWelfareCheck] = useState(false);
  const [cancellingWelfareCheck, setCancellingWelfareCheck] = useState<string | null>(null);
  const [meetings, setMeetings] = useState<HospitalMeetingRecord[]>([]);
  const [schedulableStaff, setSchedulableStaff] = useState<PatientSchedulableStaffRecord[]>([]);
  const [showScheduleForm, setShowScheduleForm] = useState(false);
  const [scheduleForm, setScheduleForm] = useState({
    hospital_id: "",
    staff_id: "",
    starts_at: "",
    duration_minutes: 30,
    timezone: resolveIANATimezone(),
    title: "",
    notes: "",
  });
  const [mutatingMeetingID, setMutatingMeetingID] = useState<string | null>(null);
  const [submittingSchedule, setSubmittingSchedule] = useState(false);
  const [bridgedSession, setBridgedSession] = useState<BridgedCallSessionRecord | null>(null);
  const [bridgedTransferBusy, setBridgedTransferBusy] = useState(false);
  const [bridgeRoomState, setBridgeRoomState] = useState<"disconnected" | "connecting" | "connected">("disconnected");
  const [bridgeCaptions, setBridgeCaptions] = useState<InterpretationSegmentMessage[]>([]);
  const [bridgedForm, setBridgedForm] = useState({
    hospital_id: "",
    staff_id: "",
    translation_enabled: true,
    language_mode: "auto",
    language_code: "es",
  });
  const bridgeRoomRef = useRef<Room | null>(null);

  const loadDashboard = useCallback(async (force = false) => {
    if (!accessToken) return;
    setLoading(true);
    setError("");
    try {
      const cacheOptions = { ttlMs: 45_000, force };
      const [profileData, consentsData, hospitalConsentsData, callsData, welfareData, auditData, meetingsData] = await Promise.all([
        cachedApiJSON<HTTPPatientProfileResponse>("patient", APIEndpoints.PATIENT_PROFILE, cacheOptions),
        cachedApiJSON<HTTPPatientConsentListResponse>("patient", APIEndpoints.PATIENT_CONSENTS, cacheOptions),
        cachedApiJSON<HTTPPatientHospitalConsentListResponse>("patient", APIEndpoints.PATIENT_HOSPITAL_CONSENTS, cacheOptions),
        cachedApiJSON<HTTPPatientCallListResponse>("patient", APIEndpoints.PATIENT_CALLS, cacheOptions),
        cachedApiJSON<HTTPPatientWelfareCheckListResponse>("patient", APIEndpoints.PATIENT_WELFARE_CHECKS, cacheOptions),
        cachedApiJSON<HTTPPatientAuditResponse>("patient", APIEndpoints.PATIENT_AUDIT, cacheOptions),
        cachedApiJSON<HTTPPatientMeetingListResponse>("patient", APIEndpoints.PATIENT_MEETINGS, cacheOptions),
      ]);

      setProfile(profileData.data ?? null);
      setConsents(consentsData.data?.consents ?? []);
      setHospitalConsents(hospitalConsentsData.data?.consents ?? []);
      setCalls(normalizePatientCalls(callsData));
      setWelfareChecks(welfareData.data?.welfare_checks ?? []);
      setAuditEvents(auditData.data?.events ?? []);
      setMeetings(meetingsData.data?.meetings ?? []);
    } catch {
      setError("Network error. Please refresh and try again.");
    } finally {
      setLoading(false);
    }
  }, [accessToken]);

  const refreshCalls = useCallback(async () => {
    if (!accessToken) return;
    try {
      const callsRes = await apiFetch("patient", APIEndpoints.PATIENT_CALLS);
      const callsData: HTTPPatientCallListResponse = await callsRes.json();
      if (callsRes.ok) {
        setCalls(normalizePatientCalls(callsData));
      }
    } catch {
      /* keep last known call list */
    }
  }, [accessToken]);

  useEffect(() => {
    if (!ready || !authenticated || !accessToken) {
      return;
    }
    void loadDashboard();
  }, [accessToken, authenticated, loadDashboard, ready, router]);

  useEffect(() => {
    if (!ready || !authenticated || !accessToken) {
      return;
    }
    preloadApiJSON<HTTPPatientCallListResponse>("patient", APIEndpoints.PATIENT_CALLS, { ttlMs: 60_000 });
    preloadApiJSON<HTTPPatientWelfareCheckListResponse>("patient", APIEndpoints.PATIENT_WELFARE_CHECKS, { ttlMs: 60_000 });
    preloadApiJSON<HTTPPatientAuditResponse>("patient", APIEndpoints.PATIENT_AUDIT, { ttlMs: 60_000 });
    preloadApiJSON<HTTPPatientMeetingListResponse>("patient", APIEndpoints.PATIENT_MEETINGS, { ttlMs: 60_000 });
  }, [accessToken, authenticated, ready]);

  const { locationPermissionBlocked, retryBrowserLocation, activeVoiceSessionId } =
    usePatientLocationSession(
      accessToken,
      consents,
      setConsents,
      setLocationNotice,
    );

  const bridgedCallSessionId = useMemo(
    () => resolveBridgedCallSessionId(activeVoiceSessionId, calls),
    [activeVoiceSessionId, calls],
  );

  const voiceCallLive = isCallInProgress(activeVoiceSessionId, calls);
  const activePatientCall = useMemo(() => findActivePatientCall(calls), [calls]);

  useEffect(() => {
    if (bridgedForm.hospital_id.trim() || hospitalConsents.length === 0) {
      return;
    }
    const first = hospitalConsents.find((c) => c.hospital_id)?.hospital_id;
    if (first) {
      setBridgedForm((current) => ({ ...current, hospital_id: first }));
    }
  }, [hospitalConsents, bridgedForm.hospital_id]);

  useEffect(() => {
    if (activeSection !== "calls" || !accessToken) {
      return;
    }
    void refreshCalls();
    const timer = window.setInterval(() => void refreshCalls(), 12_000);
    return () => window.clearInterval(timer);
  }, [activeSection, accessToken, refreshCalls]);

  useEffect(() => {
    if (activeVoiceSessionId) {
      void refreshCalls();
    }
  }, [activeVoiceSessionId, refreshCalls]);

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
    async (wsUrl?: string, token?: string) => {
      if (!wsUrl || !token) return;
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
        await room.connect(wsUrl, token);
        setBridgeRoomState("connected");
      } catch {
        setError("Interpreter started, but companion captions could not join.");
        await disconnectBridgeRoom();
      }
    },
    [disconnectBridgeRoom],
  );

  useEffect(() => () => {
    void disconnectBridgeRoom();
  }, [disconnectBridgeRoom]);

  const requestBridgedTransfer = async () => {
    if (!accessToken) return;
    setError("");
    if (!bridgedCallSessionId) {
      setError(
        "Start a phone call with Zorba and complete verification first. This page will detect your active call automatically.",
      );
      return;
    }
    if (!bridgedForm.hospital_id.trim()) {
      setError(
        "Choose a hospital under Consents first, then return here to request an interpreter.",
      );
      return;
    }
    setBridgedTransferBusy(true);
    try {
      const payload: RequestBridgedCallTransferRequest = {
        session_id: bridgedCallSessionId,
        room_sid: bridgedCallSessionId,
        hospital_id: bridgedForm.hospital_id.trim(),
        staff_id: bridgedForm.staff_id.trim() || undefined,
        transfer_reason: "Patient requested live interpretation",
      };
      const response = await apiFetch("patient", APIEndpoints.PATIENT_BRIDGED_CALL_TRANSFER, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data: HTTPBridgedCallSessionResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to request bridged transfer.");
        return;
      }
      setBridgedSession(data.data?.session ?? null);
      setBridgeCaptions([]);
      if (data.data?.patient_room_token && data.data?.livekit_ws_url) {
        await joinBridgeRoom(data.data.livekit_ws_url, data.data.patient_room_token);
      }
    } catch {
      setError("Network error while requesting bridged transfer.");
    } finally {
      setBridgedTransferBusy(false);
    }
  };

  const refreshBridgedSession = async () => {
    if (!accessToken || !bridgedCallSessionId) {
      setError(
        "No active phone call detected. Start a verified call with Zorba first.",
      );
      return;
    }
    setError("");
    try {
      const response = await apiFetch(
        "patient",
        `${APIEndpoints.PATIENT_BRIDGED_CALL_SESSION}?session_id=${encodeURIComponent(bridgedCallSessionId)}`,
      );
      const data: HTTPBridgedCallSessionResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to load bridged call session.");
        return;
      }
      setBridgedSession(data.data?.session ?? null);
      if (data.data?.patient_room_token && data.data?.livekit_ws_url) {
        await joinBridgeRoom(data.data.livekit_ws_url, data.data.patient_room_token);
      }
    } catch {
      setError("Network error while loading bridged session.");
    }
  };

  const updateBridgedTranslation = async () => {
    if (!accessToken || !bridgedCallSessionId) {
      setError(
        "No active phone call detected. Start a verified call with Zorba first.",
      );
      return;
    }
    setError("");
    try {
      const payload: UpdateBridgedCallTranslationRequest = {
        session_id: bridgedCallSessionId,
        participant: "patient",
        translation: {
          enabled: bridgedForm.translation_enabled,
          language_mode: bridgedForm.language_mode,
          language_code: bridgedForm.language_code.trim().toLowerCase(),
        },
      };
      const response = await apiFetch("patient", APIEndpoints.PATIENT_BRIDGED_CALL_TRANSLATION, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data: HTTPBridgedCallSessionResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to update translation preferences.");
        return;
      }
      setBridgedSession(data.data?.session ?? null);
    } catch {
      setError("Network error while updating translation preferences.");
    }
  };

  const endBridgedCall = async () => {
    if (!accessToken || !bridgedCallSessionId) {
      setError(
        "No active phone call detected. Start a verified call with Zorba first.",
      );
      return;
    }
    setError("");
    try {
      const response = await apiFetch("patient", APIEndpoints.PATIENT_BRIDGED_CALL_END, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          session_id: bridgedCallSessionId,
          reason: "Ended by patient",
        }),
      });
      const data: HTTPBridgedCallSessionResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to end bridged call.");
        return;
      }
      setBridgedSession(data.data?.session ?? null);
      setBridgeCaptions([]);
      await disconnectBridgeRoom();
    } catch {
      setError("Network error while ending bridged call.");
    }
  };

  const consentState = useMemo(() => {
    const map = new Map<string, ConsentRecord>();
    for (const consent of consents) {
      if (consent.consent_type && !map.has(consent.consent_type)) {
        map.set(consent.consent_type, consent);
      }
    }
    return map;
  }, [consents]);

  const activeConsents = useMemo(
    () => consents.filter((item) => item.status === "active").length,
    [consents],
  );

  const mutateConsent = async (consentType: string, enabled: boolean) => {
    if (!accessToken) return;
    setMutatingConsent(consentType);
    setError("");
    const payload: HTTPPatientConsentMutationRequest = {
      consent_type: consentType,
      source: "patient-web-portal",
    };

    try {
      const response = await apiFetch("patient", APIEndpoints.PATIENT_CONSENTS, {
        method: enabled ? "POST" : "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data: HTTPPatientConsentMutationResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Failed to update consent.");
        return;
      }

      setConsents((current) => {
        const next = current.filter(
          (item) => item.consent_type !== data.data?.consent?.consent_type,
        );
        if (data.data?.consent) next.unshift(data.data.consent);
        return next;
      });
      clearApiCache("patient");
    } catch {
      setError("Network error while updating consent.");
    } finally {
      setMutatingConsent(null);
    }
  };

  const createWelfareCheck = async (event: React.FormEvent) => {
    event.preventDefault();
    if (!accessToken || creatingWelfareCheck) return;

    const scheduledAt = new Date(welfareScheduledAt);
    if (Number.isNaN(scheduledAt.getTime())) {
      setError("Choose a valid date and time.");
      return;
    }
    if (welfareDetail.length > 1000) {
      setError("Reason detail must be 1000 characters or less.");
      return;
    }

    setCreatingWelfareCheck(true);
    setError("");
    const timezone = resolveIANATimezone();
    const payload: HTTPPatientWelfareCheckCreateRequest = {
      scheduled_at: scheduledAt.toISOString(),
      timezone,
      reason_code: welfareReason,
      reason_detail: welfareDetail.trim(),
    };

    try {
      const response = await apiFetch("patient", APIEndpoints.PATIENT_WELFARE_CHECKS, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data: HTTPPatientWelfareCheckResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to schedule welfare check.");
        return;
      }
      const created = data.data?.welfare_check;
      if (created) {
        setWelfareChecks((current) => [created, ...current]);
      }
      clearApiCache("patient");
      setWelfareDetail("");
      setWelfareScheduledAt(localDateTimeValue());
      toast.success("Welfare check scheduled.");
    } catch {
      setError("Network error while scheduling welfare check.");
    } finally {
      setCreatingWelfareCheck(false);
    }
  };

  const cancelWelfareCheck = async (id?: string) => {
    if (!accessToken || !id) return;

    setCancellingWelfareCheck(id);
    setError("");
    try {
      const response = await apiFetch(
        "patient",
        `${APIEndpoints.PATIENT_WELFARE_CHECKS}/${encodeURIComponent(id)}`,
        { method: "DELETE" },
      );
      const data: HTTPPatientWelfareCheckResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to cancel welfare check.");
        return;
      }
      const cancelled = data.data?.welfare_check;
      if (cancelled) {
        setWelfareChecks((current) =>
          current.map((item) => (item.id === id ? cancelled : item)),
        );
      }
      clearApiCache("patient");
      toast.success("Welfare check cancelled.");
    } catch {
      setError("Network error while cancelling welfare check.");
    } finally {
      setCancellingWelfareCheck(null);
    }
  };

  const askHealthQuestion = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!accessToken || !question.trim()) return;

    setAskingQuestion(true);
    setError("");
    setAnswer("");
    setAnswerSources([]);
    setRecordsNotice("");

    const payload: HTTPPatientHealthAnswerRequest = {
      question: question.trim(),
      top_k: 5,
    };

    try {
      const response = await apiFetch("patient", APIEndpoints.PATIENT_RECORDS_ANSWER, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data: HTTPPatientHealthAnswerResponse = await response.json();
      if (!response.ok) {
        if (data.error?.code === "NO_HEALTH_RECORDS") {
          setRecordsNotice(
            data.error.message ||
              "No health records are available yet. Once records are added, Zorba can answer questions with citations.",
          );
          return;
        }
        setError(data.error?.message || "Unable to answer question.");
        return;
      }
      setAnswer(data.data?.answer ?? "No answer returned.");
      setAnswerSources(
        (data.data?.citations ?? [])
          .map((citation) => citation.source_file)
          .filter((value): value is string => Boolean(value)),
      );
    } catch {
      setError("Network error while asking your question.");
    } finally {
      setAskingQuestion(false);
    }
  };

  const loadSchedulableStaff = useCallback(async (hospitalID: string) => {
    if (!accessToken || !hospitalID) return;
    try {
      const data = await cachedApiJSON<HTTPPatientSchedulableStaffResponse>(
        "patient",
        `${APIEndpoints.PATIENT_SCHEDULABLE_STAFF}?hospital_id=${encodeURIComponent(hospitalID)}`,
        { ttlMs: 5 * 60_000 },
      );
      const staff = data.data?.staff ?? [];
      setSchedulableStaff(staff);
      setScheduleForm((prev) => {
        if (!prev.staff_id && staff.length) {
          return { ...prev, staff_id: staff[0].staff_id };
        }
        return prev;
      });
    } catch {
      // best-effort
    }
  }, [accessToken]);

  const handleScheduleMeeting = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!accessToken) return;
    const timezone = resolveIANATimezone(scheduleForm.timezone);
    if (timezone !== scheduleForm.timezone) {
      setScheduleForm((prev) => ({ ...prev, timezone }));
    }
    setSubmittingSchedule(true);
    try {
      const payload: HTTPPatientMeetingScheduleRequest = {
        staff_id: scheduleForm.staff_id,
        hospital_id: scheduleForm.hospital_id,
        starts_at: new Date(scheduleForm.starts_at).toISOString(),
        duration_minutes: scheduleForm.duration_minutes,
        timezone,
        title: scheduleForm.title || undefined,
        notes: scheduleForm.notes || undefined,
      };
      const res = await apiFetch("patient", APIEndpoints.PATIENT_MEETINGS, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data: HTTPPatientMeetingMutationResponse = await res.json();
      if (!res.ok) {
        setError(data.error?.message || "Unable to schedule meeting.");
        return;
      }
      if (data.data?.meeting) {
        setMeetings((prev) => [data.data!.meeting!, ...prev]);
      }
      clearApiCache("patient");
      setShowScheduleForm(false);
      setScheduleForm((prev) => ({
        ...prev,
        staff_id: "",
        starts_at: "",
        title: "",
        notes: "",
      }));
    } catch {
      setError("Network error while scheduling meeting.");
    } finally {
      setSubmittingSchedule(false);
    }
  };

  const handleCancelMeeting = async (meetingID: string) => {
    if (!accessToken) return;
    setMutatingMeetingID(meetingID);
    try {
      const res = await apiFetch("patient", `${APIEndpoints.PATIENT_MEETINGS}/${encodeURIComponent(meetingID)}`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reason: "Cancelled by patient" }),
      });
      const data: HTTPPatientMeetingMutationResponse = await res.json();
      if (!res.ok) {
        setError(data.error?.message || "Unable to cancel meeting.");
        return;
      }
      clearApiCache("patient");
      setMeetings((prev) =>
        prev.map((m) =>
          m.id === meetingID
            ? data.data?.meeting ?? { ...m, status: "cancelled" }
            : m,
        ),
      );
    } catch {
      setError("Network error while cancelling meeting.");
    } finally {
      setMutatingMeetingID(null);
    }
  };

  if (!ready || loading) {
    return (
      <main className="min-h-screen bg-slate-100 px-5 py-8 dark:bg-slate-950">
        <div className="mx-auto max-w-7xl space-y-6">
          <div className="h-28 rounded-2xl bg-white shadow-sm dark:bg-slate-900" />
          <div className="grid gap-4 md:grid-cols-3">
            <div className="h-32 rounded-2xl bg-white shadow-sm dark:bg-slate-900" />
            <div className="h-32 rounded-2xl bg-white shadow-sm dark:bg-slate-900" />
            <div className="h-32 rounded-2xl bg-white shadow-sm dark:bg-slate-900" />
          </div>
          <div className="hidden">
            <div className="h-80 rounded-2xl bg-white shadow-sm dark:bg-slate-900" />
            <div className="h-80 rounded-2xl bg-white shadow-sm dark:bg-slate-900" />
          </div>
        </div>
      </main>
    );
  }

  const lastAudit = formatDateTime(auditEvents[0]?.timestamp, "No audits yet");

  return (
    <DashboardShell
      title="Patient Portal"
      subtitle="Consent-aware care workspace"
      navItems={sectionNavItems}
      activeSection={activeSection}
      onSectionChange={setActiveSection}
      onLogout={() => {
        void logoutAuth("patient").then(() => router.push("/login/patient"));
      }}
    >
      <PageHeader
        eyebrow="Patient Dashboard"
        title={profile?.full_name || "Patient Portal"}
        description="Manage AI permissions, query health records, review calls, and verify every clinical access event."
        actions={
          <Button
            type="button"
            variant="outline"
            onClick={() => {
              toast.message("Refreshing dashboard...");
              void loadDashboard(true);
            }}
          >
            <RefreshCw className="h-4 w-4" />
            Refresh
          </Button>
        }
      />

      {error ? <StatusBanner tone="error" message={error} /> : null}

          {activeSection === "dashboard" ? (
            <section className="space-y-6">
              <PeakPatientHome profile={profile} meetings={meetings} hospitalConsents={hospitalConsents} auditEvents={auditEvents} activeConsents={activeConsents} onNavigate={setActiveSection} />

              <div className="hidden">
                <StatCard
                  icon={FileText}
                  label="Active Consents"
                  value={activeConsents}
                  trend={`${consents.length} total consent records`}
                />
                <StatCard
                  icon={Phone}
                  label="Total Calls"
                  value={calls.length}
                  tone="orange"
                  trend={profile?.voice_phone || "+1 (318) 516-2690"}
                />
                <StatCard
                  icon={HeartPulse}
                  label="Scheduled Checks"
                  value={welfareChecks.filter((check) => ["scheduled", "pending"].includes((check.status || "").toLowerCase())).length}
                  tone="emerald"
                  trend={`${welfareChecks.length} total welfare checks`}
                />
                <StatCard
                  icon={Shield}
                  label="Last Audit"
                  value={auditEvents.length ? "Logged" : "None"}
                  tone="emerald"
                  trend={lastAudit}
                />
              </div>

              <div className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
                <section className="clinical-card p-6">
                  <h2 className="flex items-center gap-2 text-lg font-black">
                    <User className="h-5 w-5 text-indigo-600" />
                    Health Card Info
                  </h2>
                  <div className="mt-6 grid gap-5 sm:grid-cols-2">
                    <InfoRow icon={Phone} label="Phone" value={profile?.phone_number} />
                    <InfoRow icon={Mail} label="Email" value={profile?.email} />
                    <InfoRow
                      icon={Calendar}
                      label="Date of Birth"
                      value={
                        profile?.date_of_birth
                          ? formatDateTime(profile.date_of_birth, "Not listed")
                          : undefined
                      }
                    />
                    <InfoRow
                      icon={Clock}
                      label="Support Window"
                      value={profile?.support_window || "24/7 Hours"}
                    />
                  </div>
                  <div className="mt-6 rounded-2xl border border-indigo-100 bg-indigo-50/70 p-4">
                    <p className="text-sm font-black text-indigo-950">
                      AI Support Hotline
                    </p>
                    <p className="mt-1 text-xs leading-5 text-indigo-800">
                      Call directly for prescription refills, symptoms, or
                      safety triage.
                    </p>
                    <a
                      href={`tel:${profile?.voice_phone || "+13185162690"}`}
                      className="mt-3 inline-flex h-10 items-center justify-center gap-2 rounded-xl bg-indigo-600 px-4 text-sm font-bold text-white shadow-glow"
                    >
                      <Phone className="h-4 w-4" /> Call Zorba ({profile?.voice_phone || "+1 (318) 516-2690"})
                    </a>
                  </div>
                </section>

                <AuditTimeline events={auditEvents.slice(0, 5)} />
              </div>
            </section>
          ) : null}

          {activeSection === "consents" ? (
            <ConsentPanel
              consentState={consentState}
              hospitalConsents={hospitalConsents}
              setHospitalConsents={setHospitalConsents}
              mutatingConsent={mutatingConsent}
              mutateConsent={mutateConsent}
              setError={setError}
            />
          ) : null}

          {activeSection === "records" ? (
            <RecordsPanel
              question={question}
              setQuestion={setQuestion}
              askHealthQuestion={askHealthQuestion}
              askingQuestion={askingQuestion}
              answer={answer}
              recordsNotice={recordsNotice}
              answerSources={answerSources}
            />
          ) : null}

          {activeSection === "calls" ? (
            <div className="space-y-6">
              <section className="clinical-card p-6">
                <h2 className="flex items-center gap-2 text-lg font-black">
                  <Languages className="h-5 w-5 text-indigo-600" />
                  Hospital interpreter
                </h2>
                <p className="mt-2 text-sm leading-7 text-slate-600">
                  While you are on a verified call with Zorba, request a hospital staff member to join with live translation. No technical IDs to copy.
                </p>

                <div
                  className={`mt-5 rounded-2xl border p-4 text-sm ${
                    voiceCallLive
                      ? "border-emerald-200 bg-emerald-50/80 text-emerald-950"
                      : "border-amber-200 bg-amber-50/80 text-amber-950"
                  }`}
                >
                  {voiceCallLive ? (
                    <>
                      <p className="font-black">Active phone call detected</p>
                      <p className="mt-1 leading-6 text-emerald-900/90">
                        {activePatientCall?.started_at
                          ? `In progress since ${formatDateTime(activePatientCall.started_at)}.`
                          : "Your call is linked — you can request an interpreter below."}
                      </p>
                    </>
                  ) : (
                    <>
                      <p className="font-black">No active call yet</p>
                      <p className="mt-1 leading-6">
                        Call the Zorba assistance line, complete phone verification on the call, then return here. This page refreshes automatically.
                      </p>
                    </>
                  )}
                </div>

                {hospitalConsents.length === 0 ? (
                  <p className="mt-4 rounded-xl border border-dashed border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-600">
                    Approve at least one hospital under <strong>Consents</strong> before requesting an interpreter.
                  </p>
                ) : (
                  <div className="mt-5 grid gap-4 md:grid-cols-2">
                    <label className="space-y-2 text-sm font-semibold text-slate-700 md:col-span-2">
                      <span>Hospital</span>
                      <select
                        className="h-11 w-full rounded-xl border border-slate-200 px-4 text-sm"
                        value={bridgedForm.hospital_id}
                        onChange={(e) =>
                          setBridgedForm((current) => ({
                            ...current,
                            hospital_id: e.target.value,
                          }))
                        }
                      >
                        <option value="">Choose a hospital</option>
                        {hospitalConsents.map((consent) => (
                          <option key={consent.hospital_id} value={consent.hospital_id}>
                            {consent.hospital_name || consent.hospital_id}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="space-y-2 text-sm font-semibold text-slate-700">
                      <span>Language you want to hear (optional)</span>
                      <input
                        className="h-11 w-full rounded-xl border border-slate-200 px-4 text-sm"
                        value={bridgedForm.language_code}
                        onChange={(e) =>
                          setBridgedForm((current) => ({
                            ...current,
                            language_code: e.target.value,
                          }))
                        }
                        placeholder="es, ne, hi…"
                      />
                    </label>
                    <label className="space-y-2 text-sm font-semibold text-slate-700">
                      <span>Preferred clinician (optional)</span>
                      <input
                        className="h-11 w-full rounded-xl border border-slate-200 px-4 text-sm"
                        value={bridgedForm.staff_id}
                        onChange={(e) =>
                          setBridgedForm((current) => ({
                            ...current,
                            staff_id: e.target.value,
                          }))
                        }
                        placeholder="Leave blank for any available staff"
                      />
                    </label>
                  </div>
                )}

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
                    {bridgedForm.translation_enabled ? "Live translation on" : "Live translation off"}
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
                    {bridgedForm.language_mode === "auto" ? "Auto-detect language" : "Use language above only"}
                  </button>
                </div>
                <div className="mt-5 flex flex-wrap gap-3">
                  <Button
                    type="button"
                    variant="healthcare"
                    disabled={
                      bridgedTransferBusy ||
                      !voiceCallLive ||
                      !bridgedForm.hospital_id.trim() ||
                      hospitalConsents.length === 0
                    }
                    onClick={() => void requestBridgedTransfer()}
                  >
                    {bridgedTransferBusy ? "Requesting…" : "Request hospital interpreter"}
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={!bridgedCallSessionId}
                    onClick={() => void refreshBridgedSession()}
                  >
                    Refresh status
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    disabled={!bridgedCallSessionId}
                    onClick={() => void updateBridgedTranslation()}
                  >
                    Save language settings
                  </Button>
                  <Button
                    type="button"
                    variant="outline"
                    disabled={!bridgedCallSessionId}
                    onClick={() => void endBridgedCall()}
                  >
                    End interpreter session
                  </Button>
                </div>
                {bridgedSession ? (
                  <div className="mt-5 rounded-2xl border border-indigo-100 bg-indigo-50/70 p-4 text-sm text-slate-700">
                    <p className="font-black text-indigo-950">
                      Status: {(bridgedSession.status || "pending").replaceAll("_", " ")}
                    </p>
                    <p className="mt-2">
                      Hospital:{" "}
                      {hospitalConsents.find((c) => c.hospital_id === bridgedSession.hospital_id)
                        ?.hospital_name ||
                        bridgedSession.hospital_id ||
                        "unknown"}
                    </p>
                    <p>
                      Clinician: {bridgedSession.staff_id ? "assigned" : "waiting for hospital to connect"}
                    </p>
                    <p>
                      Translation:{" "}
                      {bridgedSession.patient_translation?.enabled ? "on" : "off"} ·{" "}
                      {bridgedSession.patient_translation?.language_mode || "auto"} ·{" "}
                      {bridgedSession.patient_translation?.language_code || "default"}
                    </p>
                    <p className="mt-2">
                      Companion captions:{" "}
                      <span className={bridgeRoomState === "connected" ? "font-bold text-emerald-700" : "font-bold text-slate-500"}>
                        {bridgeRoomState}
                      </span>
                    </p>
                    <p className="mt-2">
                      Interpreter mode:{" "}
                      <span className="font-bold text-indigo-900">
                        {bridgedSession.status === "connected" ? "live" : "waiting for clinician"}
                      </span>
                    </p>
                  </div>
                ) : null}
                {bridgeRoomState !== "disconnected" || bridgeCaptions.length > 0 ? (
                  <div className="mt-5 rounded-2xl border border-emerald-100 bg-emerald-50/60 p-4">
                    <p className="text-sm font-black text-emerald-950">Live interpreter captions</p>
                    <p className="mt-1 text-xs font-semibold text-emerald-800">
                      Audio continues on your phone call. This page only mirrors interpreter captions and connection status.
                    </p>
                    {bridgeCaptions.length === 0 ? (
                      <p className="mt-2 text-sm text-slate-500">Waiting for the patient or clinician to speak...</p>
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
              <CallsPanel calls={calls} activeSessionId={bridgedCallSessionId} />
            </div>
          ) : null}

          {activeSection === "audit" ? (
            <AuditTable events={auditEvents} />
          ) : null}

          {activeSection === "meetings" ? (
            <PatientMeetingPanel
              meetings={meetings}
              hospitalConsents={hospitalConsents}
              schedulableStaff={schedulableStaff}
              showScheduleForm={showScheduleForm}
              scheduleForm={scheduleForm}
              setScheduleForm={setScheduleForm}
              setShowScheduleForm={setShowScheduleForm}
              mutatingMeetingID={mutatingMeetingID}
              submittingSchedule={submittingSchedule}
              onRefresh={() => void loadDashboard()}
              onSchedule={(e) => void handleScheduleMeeting(e)}
              onCancel={(id) => void handleCancelMeeting(id)}
              onHospitalChange={(hospitalID) => void loadSchedulableStaff(hospitalID)}
            />
          ) : null}

          {activeSection === "appointments" ? (
            <PatientAppointmentBookingPanel
              hospitals={hospitalConsents
                .filter((c) => c.hospital_id && !c.revoked_at)
                .map((c) => ({
                  hospital_id: c.hospital_id!,
                  hospital_name: c.hospital_name || c.hospital_id!,
                }))}
              loadStaff={async (hospitalID) => {
                const res = await apiFetch(
                  "patient",
                  `${APIEndpoints.PATIENT_SCHEDULABLE_STAFF}?hospital_id=${encodeURIComponent(hospitalID)}`,
                );
                const data: HTTPPatientSchedulableStaffResponse = await res.json();
                return (data.data?.staff ?? []).map((s) => ({
                  staff_id: s.staff_id,
                  name: s.name || s.staff_id,
                  role: s.role || "staff",
                }));
              }}
            />
          ) : null}

          {activeSection === "welfare" ? (
            <WelfarePanel
              welfareChecks={welfareChecks}
              welfareScheduledAt={welfareScheduledAt}
              welfareReason={welfareReason}
              welfareDetail={welfareDetail}
              creatingWelfareCheck={creatingWelfareCheck}
              cancellingWelfareCheck={cancellingWelfareCheck}
              setWelfareScheduledAt={setWelfareScheduledAt}
              setWelfareReason={setWelfareReason}
              setWelfareDetail={setWelfareDetail}
              onCreate={(event) => void createWelfareCheck(event)}
              onCancel={(id) => void cancelWelfareCheck(id)}
            />
          ) : null}

          {activeSection === "gps" ? (
            <section className="clinical-card p-6">
              <h2 className="flex items-center gap-2 text-lg font-black">
                <Compass className="h-5 w-5 text-indigo-600" />
                Emergency GPS Channel
              </h2>
              <p className="mt-2 text-sm leading-7 text-slate-600">
                {locationNotice ||
                  "Location streaming stays idle until an emergency voice session requests it."}
              </p>
              {locationPermissionBlocked ? (
                <Button
                  type="button"
                  variant="healthcare"
                  className="mt-5"
                  onClick={() => retryBrowserLocation()}
                >
                  Enable GPS Location
                </Button>
              ) : (
                <div className="mt-5 inline-flex items-center gap-2 rounded-full bg-emerald-50 px-3 py-1.5 text-xs font-black text-emerald-700 ring-1 ring-emerald-100">
                  <span className="h-2 w-2 rounded-full bg-emerald-500" />
                  Permission ready
                </div>
              )}
            </section>
          ) : null}
    </DashboardShell>
  );
}

export default function PatientHomePage() {
  return (
    <Suspense
      fallback={
        <main className="mesh-bg min-h-screen px-5 py-8">
          <div className="mx-auto max-w-7xl space-y-6">
            <div className="h-28 rounded-[2rem] bg-white/80 dark:bg-slate-900/80" />
            <div className="grid gap-4 md:grid-cols-3">
              <div className="h-32 rounded-3xl bg-white/80 dark:bg-slate-900/80" />
              <div className="h-32 rounded-3xl bg-white/80 dark:bg-slate-900/80" />
              <div className="h-32 rounded-3xl bg-white/80 dark:bg-slate-900/80" />
            </div>
          </div>
        </main>
      }
    >
      <PatientHomePageContent />
    </Suspense>
  );
}

function PeakPatientHome({
  profile,
  meetings,
  hospitalConsents,
  auditEvents,
  activeConsents,
  onNavigate,
}: {
  profile: HTTPPatientProfileResponse["data"] | null;
  meetings: HospitalMeetingRecord[];
  hospitalConsents: PatientHospitalConsentRecord[];
  auditEvents: AuditEventRecord[];
  activeConsents: number;
  onNavigate: (section: string) => void;
}) {
  const upcomingMeeting = meetings.find((meeting) => meeting.status !== "cancelled") ?? null;
  const pendingConsentCount = hospitalConsents.filter((consent) => consent.status === "pending").length;
  const nextAction = upcomingMeeting
    ? "Your next visit is ready to join."
    : pendingConsentCount > 0
      ? "Review a hospital consent request."
      : "Ask a question about your records.";

  return (
    <section className="space-y-4" aria-label="Next best action">
      <div className="rounded-[var(--zh-radius-panel)] border border-indigo-100 bg-white p-6 shadow-sm dark:border-slate-700 dark:bg-slate-900">
        <div className="flex flex-col gap-5 lg:flex-row lg:items-center lg:justify-between">
          <div className="max-w-2xl">
            <p className="text-base font-bold text-indigo-700 dark:text-indigo-300">Good to see you, {profile?.full_name || "there"}</p>
            <h2 className="mt-2 font-patient text-[length:var(--zh-display-size)] font-semibold leading-tight text-slate-950 dark:text-white">{nextAction}</h2>
            <p className="mt-3 text-base leading-6 text-slate-600 dark:text-slate-300">Your care, records, and privacy controls are in one place. Choose what needs your attention now.</p>
          </div>
          <div className="flex flex-wrap gap-3">
            {upcomingMeeting ? (
              <Button type="button" variant="healthcare" onClick={() => onNavigate("meetings")} className="min-h-[44px]">Join visit</Button>
            ) : pendingConsentCount > 0 ? (
              <Button type="button" variant="healthcare" onClick={() => onNavigate("consents")} className="min-h-[44px]">Review consent</Button>
            ) : (
              <Button type="button" variant="healthcare" onClick={() => onNavigate("records")} className="min-h-[44px]">Ask records</Button>
            )}
            <Button type="button" variant="outline" onClick={() => onNavigate("welfare")} className="min-h-[44px] border-rose-200 text-rose-700 hover:bg-rose-50">Emergency / welfare</Button>
          </div>
        </div>
      </div>
      <div className="grid gap-4 lg:grid-cols-[1.35fr_0.65fr]">
        <article className="rounded-[var(--zh-radius-card)] border border-slate-200 bg-white p-5 dark:border-slate-700 dark:bg-slate-900">
          <p className="text-sm font-bold uppercase tracking-wide text-slate-500">Upcoming care</p>
          {upcomingMeeting ? (
            <div className="mt-3 flex items-start justify-between gap-4">
              <div><h3 className="text-lg font-bold text-slate-950 dark:text-white">{upcomingMeeting.title || "Care visit"}</h3><p className="mt-1 text-base text-slate-600 dark:text-slate-300">{formatDateTime(upcomingMeeting.starts_at, "Time to be confirmed")} · {upcomingMeeting.timezone || "your local time"}</p></div>
              <span className="rounded-full bg-emerald-50 px-3 py-1 text-sm font-bold text-emerald-700">{upcomingMeeting.status || "scheduled"}</span>
            </div>
          ) : <p className="mt-3 text-base text-slate-600 dark:text-slate-300">No upcoming visits. Book a time with a linked hospital.</p>}
        </article>
        <article className="rounded-[var(--zh-radius-card)] border border-slate-200 bg-white p-5 dark:border-slate-700 dark:bg-slate-900">
          <p className="text-sm font-bold uppercase tracking-wide text-slate-500">Privacy at a glance</p>
          <p className="mt-3 text-3xl font-bold text-slate-950 dark:text-white">{activeConsents}</p>
          <p className="text-base text-slate-600 dark:text-slate-300">active consent grants</p>
          <button type="button" className="mt-4 min-h-[44px] text-base font-bold text-indigo-700 underline-offset-4 hover:underline dark:text-indigo-300" onClick={() => onNavigate("consents")}>Manage privacy</button>
        </article>
      </div>
      <div className="rounded-[var(--zh-radius-card)] border border-slate-200 bg-slate-50 p-5 dark:border-slate-700 dark:bg-slate-800">
        <p className="text-sm font-bold uppercase tracking-wide text-slate-500">Recent care events</p>
        {auditEvents.length === 0 ? <p className="mt-3 text-base text-slate-600 dark:text-slate-300">No care events yet.</p> : <ul className="mt-3 space-y-2">{auditEvents.slice(0, 3).map((event) => <li key={`-`} className="flex items-center justify-between gap-4 text-base"><span className="text-slate-700 dark:text-slate-200">{formatEventType(event.event_type)}</span><time className="text-sm text-slate-500">{formatDateTime(event.timestamp, "Recently")}</time></li>)}</ul>}
      </div>
    </section>
  );
}

function InfoRow({
  icon: Icon,
  label,
  value,
}: {
  icon: typeof Phone;
  label: string;
  value?: string;
}) {
  return (
    <div className="flex items-start gap-3">
      <Icon className="mt-0.5 h-4 w-4 text-slate-400" />
      <div>
        <p className="text-xs font-black uppercase tracking-wide text-slate-400">
          {label}
        </p>
        <p className="mt-1 text-sm font-bold text-slate-800 break-words">
          {value || "Not listed"}
        </p>
      </div>
    </div>
  );
}

function ConsentPanel({
  consentState,
  hospitalConsents,
  setHospitalConsents,
  mutatingConsent,
  mutateConsent,
  setError,
}: {
  consentState: Map<string, ConsentRecord>;
  hospitalConsents: PatientHospitalConsentRecord[];
  setHospitalConsents: React.Dispatch<React.SetStateAction<PatientHospitalConsentRecord[]>>;
  mutatingConsent: string | null;
  mutateConsent: (consentType: string, enabled: boolean) => void;
  setError: (value: string) => void;
}) {
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const scanTimerRef = useRef<number | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const [scannerOpen, setScannerOpen] = useState(false);
  const [manualToken, setManualToken] = useState("");
  const [pendingRequest, setPendingRequest] =
    useState<HospitalConsentRequestRecord | null>(null);
  const [scannerNotice, setScannerNotice] = useState("");
  const [mutatingHospitalID, setMutatingHospitalID] = useState<string | null>(null);

  const stopScanner = useCallback(() => {
    if (scanTimerRef.current) {
      window.clearInterval(scanTimerRef.current);
      scanTimerRef.current = null;
    }
    streamRef.current?.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
    setScannerOpen(false);
  }, []);

  useEffect(() => stopScanner, [stopScanner]);

  const lookupToken = async (rawToken: string) => {
    const token = extractConsentToken(rawToken);
    if (!token) return;
    setError("");
    setScannerNotice("");
    try {
      const response = await apiFetch(
        "patient",
        `${APIEndpoints.PATIENT_CONSENT_REQUESTS}/${encodeURIComponent(token)}`,
      );
      const data: HTTPPatientConsentRequestLookupResponse = await response.json();
      if (!response.ok || !data.data?.request) {
        setError(data.error?.message || "Unable to load consent request.");
        return;
      }
      setPendingRequest(data.data.request);
      setManualToken(token);
      stopScanner();
    } catch {
      setError("Network error while loading consent request.");
    }
  };

  const startScanner = async () => {
    setError("");
    setScannerNotice("");
    if (!("BarcodeDetector" in window)) {
      setScannerNotice("Camera scanning is not supported in this browser. Paste the QR token instead.");
      return;
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({
        video: { facingMode: "environment" },
      });
      streamRef.current = stream;
      setScannerOpen(true);
      window.setTimeout(() => {
        if (videoRef.current) {
          videoRef.current.srcObject = stream;
          void videoRef.current.play();
        }
      }, 0);
      const detector = new (window as unknown as {
        BarcodeDetector: new (config: { formats: string[] }) => {
          detect: (source: HTMLVideoElement) => Promise<Array<{ rawValue: string }>>;
        };
      }).BarcodeDetector({ formats: ["qr_code"] });
      scanTimerRef.current = window.setInterval(async () => {
        if (!videoRef.current) return;
        const codes = await detector.detect(videoRef.current).catch(() => []);
        if (codes[0]?.rawValue) void lookupToken(codes[0].rawValue);
      }, 700);
    } catch {
      setScannerNotice("Camera permission is required to scan. Paste the QR token instead.");
    }
  };

  const approveRequest = async () => {
    const token = pendingRequest?.token || manualToken;
    if (!token) return;
    setError("");
    try {
      const response = await apiFetch(
        "patient",
        `${APIEndpoints.PATIENT_CONSENT_REQUESTS}/${encodeURIComponent(token)}/approve`,
        { method: "POST" },
      );
      const data: HTTPPatientConsentRequestApproveResponse = await response.json();
      if (!response.ok || !data.data?.consent) {
        setError(data.error?.message || "Unable to approve hospital consent.");
        return;
      }
      const approvedConsent = data.data.consent;
      setHospitalConsents((current) => {
        const next = current.filter(
          (item) => item.hospital_id !== approvedConsent.hospital_id,
        );
        return [approvedConsent, ...next];
      });
      setPendingRequest(null);
      setManualToken("");
      setScannerNotice("Hospital consent granted.");
    } catch {
      setError("Network error while approving hospital consent.");
    }
  };

  const revokeHospitalConsent = async (hospitalID?: string) => {
    if (!hospitalID) return;
    setMutatingHospitalID(hospitalID);
    setError("");
    try {
      const response = await apiFetch(
        "patient",
        `${APIEndpoints.PATIENT_HOSPITAL_CONSENTS}/${encodeURIComponent(hospitalID)}`,
        { method: "DELETE" },
      );
      const data = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to revoke hospital consent.");
        return;
      }
      setHospitalConsents((current) =>
        current.map((item) =>
          item.hospital_id === hospitalID
            ? { ...item, status: "revoked", revoked_at: new Date().toISOString() }
            : item,
        ),
      );
    } catch {
      setError("Network error while revoking hospital consent.");
    } finally {
      setMutatingHospitalID(null);
    }
  };

  return (
    <div className="space-y-6">
      <section className="clinical-card p-6">
        <h2 className="flex items-center gap-2 text-lg font-black">
          <ToggleLeft className="h-5 w-5 text-indigo-600" />
          Consent Center
        </h2>
        <p className="mt-2 text-sm leading-7 text-slate-600">
          Toggle AI processing privileges and approve hospital access from staff QR requests.
        </p>
        <div className="mt-6 grid gap-4 md:grid-cols-2">
          {consentTypes.map((consentType) => {
            const current = consentState.get(consentType);
            const active = current?.status === "active";
            return (
              <div
                key={consentType}
                className={`rounded-2xl border p-4 transition-all ${
                  active
                    ? "border-indigo-200 bg-indigo-50/60"
                    : "border-slate-200 bg-white"
                }`}
              >
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <p className="text-sm font-black text-slate-950">
                      {consentLabels[consentType]}
                    </p>
                    <p className="mt-1 text-xs font-bold uppercase tracking-wide text-slate-400">
                      {current?.status || "inactive"}
                    </p>
                  </div>
                  <button
                    type="button"
                    disabled={mutatingConsent === consentType}
                    onClick={() => mutateConsent(consentType, !active)}
                    className={`relative h-8 w-14 rounded-full p-1 transition-all ${
                      active ? "bg-indigo-600" : "bg-slate-300"
                    }`}
                  >
                    <span
                      className={`block h-6 w-6 rounded-full bg-white shadow transition-transform ${
                        active ? "translate-x-6" : "translate-x-0"
                      }`}
                    />
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      </section>

      <section className="clinical-card p-6">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h2 className="flex items-center gap-2 text-lg font-black">
              <QrCode className="h-5 w-5 text-indigo-600" />
              Hospital QR Consent
            </h2>
            <p className="mt-2 text-sm leading-7 text-slate-600">
              Scan a hospital staff QR code, review the request, then approve access.
            </p>
          </div>
          <Button type="button" variant="healthcare" onClick={() => void startScanner()}>
            <Camera className="h-4 w-4" />
            Scan QR
          </Button>
        </div>

        {scannerOpen ? (
          <div className="mt-5 overflow-hidden rounded-2xl border border-slate-200 bg-slate-950">
            <video ref={videoRef} className="aspect-video w-full object-cover" muted playsInline />
          </div>
        ) : null}

        {scannerNotice ? (
          <div className="mt-5 rounded-xl border border-indigo-100 bg-indigo-50/80 p-4 text-sm font-semibold text-indigo-700">
            {scannerNotice}
          </div>
        ) : null}

        <form
          onSubmit={(event) => {
            event.preventDefault();
            void lookupToken(manualToken);
          }}
          className="mt-5 flex flex-col gap-3 sm:flex-row"
        >
          <input
            value={manualToken}
            onChange={(event) => setManualToken(event.target.value)}
            placeholder="Paste QR token"
            className="h-11 min-w-0 flex-1 rounded-xl border border-slate-200 bg-white px-4 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
          />
          <Button type="submit" variant="outline">
            Lookup
          </Button>
        </form>

        {pendingRequest ? (
          <div className="mt-5 rounded-2xl border border-indigo-200 bg-indigo-50/70 p-5">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <p className="text-xs font-black uppercase tracking-wide text-indigo-700">
                  Pending request
                </p>
                <h3 className="mt-2 text-lg font-black text-slate-950">
                  {pendingRequest.hospital_name}
                </h3>
                <p className="mt-1 text-sm font-semibold text-slate-600">
                  Requested by {pendingRequest.staff_name || "hospital staff"}
                </p>
                {pendingRequest.note ? (
                  <p className="mt-3 text-sm leading-6 text-slate-600">
                    {pendingRequest.note}
                  </p>
                ) : null}
                <div className="mt-4 flex flex-wrap gap-2">
                  {(pendingRequest.requested_permissions ?? []).map((permission) => (
                    <span
                      key={permission}
                      className="rounded-full bg-white px-3 py-1 text-xs font-black text-indigo-700"
                    >
                      {permission.replaceAll("_", " ")}
                    </span>
                  ))}
                </div>
              </div>
              <Button type="button" variant="healthcare" onClick={() => void approveRequest()}>
                <CheckCircle className="h-4 w-4" />
                Approve
              </Button>
            </div>
          </div>
        ) : null}
      </section>

      <section className="clinical-card p-6">
        <h2 className="flex items-center gap-2 text-lg font-black">
          <Shield className="h-5 w-5 text-emerald-600" />
          Hospital Access
        </h2>
        <div className="mt-5 grid gap-3">
          {hospitalConsents.map((consent) => (
            <div
              key={consent.hospital_id}
              className="flex flex-col gap-3 rounded-2xl border border-slate-200 bg-white p-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div>
                <p className="text-sm font-black text-slate-950">
                  {consent.hospital_name || consent.hospital_id}
                </p>
                <p className="mt-1 text-xs font-semibold uppercase tracking-wide text-slate-400">
                  {consent.status || "active"} since {formatDateTime(consent.granted_at)}
                </p>
              </div>
              {consent.status !== "revoked" ? (
                <Button
                  type="button"
                  variant="outline"
                  disabled={mutatingHospitalID === consent.hospital_id}
                  onClick={() => void revokeHospitalConsent(consent.hospital_id)}
                >
                  <Trash2 className="h-4 w-4" />
                  Revoke
                </Button>
              ) : null}
            </div>
          ))}
          {hospitalConsents.length === 0 ? (
            <div className="rounded-2xl border border-dashed border-slate-200 bg-slate-50 px-4 py-10 text-center text-sm font-semibold text-slate-400">
              No hospitals have access to your patient record.
            </div>
          ) : null}
        </div>
      </section>
    </div>
  );
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

function RecordsPanel({
  question,
  setQuestion,
  askHealthQuestion,
  askingQuestion,
  answer,
  recordsNotice,
  answerSources,
}: {
  question: string;
  setQuestion: (value: string) => void;
  askHealthQuestion: (e: React.FormEvent) => void;
  askingQuestion: boolean;
  answer: string;
  recordsNotice: string;
  answerSources: string[];
}) {
  return (
    <section className="clinical-card p-6">
      <h2 className="flex items-center gap-2 text-lg font-black">
        <MessageSquare className="h-5 w-5 text-indigo-600" />
        Ask Your Records
      </h2>
      <div className="mt-6 grid gap-6 xl:grid-cols-[0.9fr_1.1fr]">
        <form onSubmit={askHealthQuestion} className="space-y-4">
          <textarea
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            placeholder="What was my last recorded dosage for blood pressure medication?"
            className="min-h-44 w-full rounded-2xl border border-slate-200 bg-slate-50/70 px-4 py-3 text-sm outline-none transition-all placeholder:text-slate-400 focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
          />
          <Button
            type="submit"
            variant="healthcare"
            className="h-11 w-full"
            disabled={askingQuestion}
          >
            <Send className="h-4 w-4" />
            {askingQuestion ? "Answering..." : "Submit Inquiry"}
          </Button>
        </form>
        <div className="rounded-2xl bg-slate-950 p-5 text-white">
          <p className="text-xs font-black uppercase tracking-wide text-indigo-200">
            Clinician Answer
          </p>
          {recordsNotice ? (
            <div className="mt-4 rounded-xl border border-orange-300/30 bg-orange-400/10 p-3 text-sm leading-6 text-orange-100">
              {recordsNotice}
            </div>
          ) : null}
          <p className="mt-4 whitespace-pre-wrap text-sm leading-7 text-slate-100">
            {answer || (recordsNotice ? "" : "Submit a question to generate an answer.")}
          </p>
          {answerSources.length > 0 ? (
            <div className="mt-5 flex flex-wrap gap-2 border-t border-white/10 pt-4">
              {answerSources.map((source) => (
                <span
                  key={source}
                  className="inline-flex items-center gap-1 rounded-full bg-white/10 px-2.5 py-1 text-xs font-bold text-slate-100"
                >
                  <CheckCircle className="h-3 w-3 text-emerald-300" />
                  {source}
                </span>
              ))}
            </div>
          ) : null}
        </div>
      </div>
    </section>
  );
}

function CallsPanel({
  calls,
  activeSessionId,
}: {
  calls: PatientCallSummary[];
  activeSessionId?: string | null;
}) {
  return (
    <section className="clinical-card p-6">
      <h2 className="flex items-center gap-2 text-lg font-black">
        <Phone className="h-5 w-5 text-indigo-600" />
        Call Timeline
      </h2>
      <div className="mt-6 space-y-4">
        {calls.map((call) => {
          const isLive =
            Boolean(activeSessionId) &&
            call.livekit_room_id === activeSessionId;
          const isActiveRow = call.status?.toLowerCase() === "active";
          return (
          <div key={call.id} className="relative rounded-2xl border border-slate-200 bg-white p-4 pl-6">
            <div className="absolute bottom-4 left-0 top-4 w-1 rounded-r-full bg-indigo-500" />
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <p className="text-sm font-black text-slate-950">
                {isLive || isActiveRow ? "Current call" : `Call #${call.id}`}
              </p>
              <span
                className={`rounded-full px-2.5 py-1 text-xs font-black ${
                  isLive || isActiveRow
                    ? "bg-emerald-100 text-emerald-800"
                    : "bg-indigo-50 text-indigo-700"
                }`}
              >
                {isLive || isActiveRow ? "live" : call.status || "completed"}
              </span>
            </div>
            <p className="mt-2 text-xs font-semibold text-slate-400">
              {formatDateTime(call.started_at, "Unavailable")}
            </p>
            <p className="mt-3 text-sm leading-6 text-slate-600">
              {call.summary || "Summary generation in progress."}
            </p>
          </div>
          );
        })}
        {calls.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-slate-200 bg-slate-50 px-4 py-10 text-center text-sm font-semibold text-slate-400">
            No call history logs recorded yet.
          </div>
        ) : null}
      </div>
    </section>
  );
}

function AuditTimeline({
  events,
}: {
  events: AuditEventRecord[];
}) {
  return (
    <section className="clinical-card p-6">
      <h2 className="flex items-center gap-2 text-lg font-black">
        <Shield className="h-5 w-5 text-indigo-600" />
        Security Audit Activity
      </h2>
      <div className="mt-6 space-y-4">
        {events.length > 0 ? (
          events.map((event, index) => (
            <div key={event.event_id || index} className="flex gap-3 text-sm">
              <div className="flex flex-col items-center">
                <div className="mt-1.5 h-2.5 w-2.5 rounded-full bg-indigo-600" />
                {index !== events.length - 1 ? (
                  <div className="mt-1 h-10 w-0.5 bg-slate-100" />
                ) : null}
              </div>
              <div>
                <p className="text-xs font-black uppercase tracking-wide text-slate-800">
                  {formatEventType(event.event_type)}
                </p>
                <p className="mt-1 text-xs text-slate-400">
                  {event.timestamp
                    ? formatTimeOnly(event.timestamp)
                    : "Recently"}{" "}
                  - {event.service_name || "Core"}
                </p>
              </div>
            </div>
          ))
        ) : (
          <p className="text-sm text-slate-400">No audits recorded yet.</p>
        )}
      </div>
    </section>
  );
}

function PatientMeetingPanel({
  meetings,
  hospitalConsents,
  schedulableStaff,
  showScheduleForm,
  scheduleForm,
  setScheduleForm,
  setShowScheduleForm,
  mutatingMeetingID,
  submittingSchedule,
  onRefresh,
  onSchedule,
  onCancel,
  onHospitalChange,
}: {
  meetings: HospitalMeetingRecord[];
  hospitalConsents: PatientHospitalConsentRecord[];
  schedulableStaff: PatientSchedulableStaffRecord[];
  showScheduleForm: boolean;
  scheduleForm: {
    hospital_id: string;
    staff_id: string;
    starts_at: string;
    duration_minutes: number;
    timezone: string;
    title: string;
    notes: string;
  };
  setScheduleForm: React.Dispatch<React.SetStateAction<{
    hospital_id: string;
    staff_id: string;
    starts_at: string;
    duration_minutes: number;
    timezone: string;
    title: string;
    notes: string;
  }>>;
  setShowScheduleForm: (v: boolean) => void;
  mutatingMeetingID: string | null;
  submittingSchedule: boolean;
  onRefresh: () => void;
  onSchedule: (e: React.FormEvent) => void;
  onCancel: (id: string) => void;
  onHospitalChange: (hospitalID: string) => void;
}) {
  const activeConsents = useMemo(
    () => hospitalConsents.filter((hc) => !hc.revoked_at),
    [hospitalConsents],
  );

  return (
    <section className="clinical-card p-6">
      <div className="flex flex-col gap-3 border-b border-slate-100 pb-5 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="flex items-center gap-2 text-lg font-black">
            <Video className="h-5 w-5 text-orange-500" />
            My Meetings
          </h2>
          <p className="mt-1 text-sm font-semibold text-slate-500">
            Schedule video visits with hospital staff.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            size="sm"
            variant="healthcare"
              disabled={!activeConsents.length}
              onClick={() => {
                if (!showScheduleForm && activeConsents.length) {
                  const first = activeConsents[0];
                setScheduleForm((prev) => ({ ...prev, hospital_id: first.hospital_id || "" }));
                onHospitalChange(first.hospital_id || "");
              }
              setShowScheduleForm(!showScheduleForm);
            }}
          >
            <Video className="h-4 w-4" />
            {showScheduleForm ? "Cancel" : "Schedule New"}
          </Button>
          <button
            type="button"
            onClick={onRefresh}
            className="inline-flex h-10 w-10 items-center justify-center rounded-xl text-slate-400 transition-colors hover:bg-slate-100 hover:text-slate-800"
          >
            <RefreshCw className="h-4 w-4" />
          </button>
        </div>
      </div>

      {showScheduleForm ? (
        <form onSubmit={onSchedule} className="mt-5 grid gap-4 rounded-2xl border border-slate-200 bg-slate-50/70 p-5 md:grid-cols-2">
          <div className="space-y-2">
            <label className="text-xs font-black uppercase tracking-wide text-slate-500">Hospital</label>
            <select
              value={scheduleForm.hospital_id}
              onChange={(e) => {
                setScheduleForm((prev) => ({ ...prev, hospital_id: e.target.value, staff_id: "" }));
                onHospitalChange(e.target.value);
              }}
              className="h-10 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
              required
            >
              <option value="">Select hospital</option>
              {activeConsents.map((hc) => (
                <option key={hc.hospital_id} value={hc.hospital_id}>
                  {hc.hospital_name || hc.hospital_id}
                </option>
              ))}
            </select>
          </div>
          <div className="space-y-2">
            <label className="text-xs font-black uppercase tracking-wide text-slate-500">Staff</label>
            <select
              value={scheduleForm.staff_id}
              onChange={(e) => setScheduleForm((prev) => ({ ...prev, staff_id: e.target.value }))}
              className="h-10 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
              required
            >
              <option value="">Select staff</option>
              {schedulableStaff.map((s) => (
                <option key={s.staff_id} value={s.staff_id}>
                  {s.name || s.staff_id} ({s.role || "staff"})
                </option>
              ))}
            </select>
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

      <div className="mt-5 grid gap-4">
        {meetings.map((meeting) => (
          <PatientMeetingCard
            key={meeting.id}
            meeting={meeting}
            mutating={mutatingMeetingID === meeting.id}
            onCancel={() => onCancel(meeting.id)}
          />
        ))}
        {meetings.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-slate-200 bg-slate-50 px-4 py-12 text-center text-sm font-semibold text-slate-400">
            No meetings scheduled.
          </div>
        ) : null}
      </div>
    </section>
  );
}

function WelfarePanel({
  welfareChecks,
  welfareScheduledAt,
  welfareReason,
  welfareDetail,
  creatingWelfareCheck,
  cancellingWelfareCheck,
  setWelfareScheduledAt,
  setWelfareReason,
  setWelfareDetail,
  onCreate,
  onCancel,
}: {
  welfareChecks: PatientWelfareCheck[];
  welfareScheduledAt: string;
  welfareReason: WelfareCheckReason;
  welfareDetail: string;
  creatingWelfareCheck: boolean;
  cancellingWelfareCheck: string | null;
  setWelfareScheduledAt: (value: string) => void;
  setWelfareReason: (value: WelfareCheckReason) => void;
  setWelfareDetail: (value: string) => void;
  onCreate: (event: React.FormEvent) => void;
  onCancel: (id?: string) => void;
}) {
  return (
    <section className="clinical-card p-6">
      <div className="border-b border-slate-100 pb-5">
        <h2 className="flex items-center gap-2 text-lg font-black">
          <HeartPulse className="h-5 w-5 text-rose-500" />
          Scheduled Welfare Checks
        </h2>
        <p className="mt-2 text-sm leading-7 text-slate-600">
          Schedule a check-in call from Zorba using your saved profile number.
        </p>
      </div>

      <form onSubmit={onCreate} className="mt-5 grid gap-4 rounded-2xl border border-slate-200 bg-slate-50/70 p-5 md:grid-cols-2">
        <label className="space-y-2">
          <span className="text-xs font-black uppercase tracking-wide text-slate-500">Date & time</span>
          <input
            type="datetime-local"
            value={welfareScheduledAt}
            onChange={(event) => setWelfareScheduledAt(event.target.value)}
            className="h-10 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
            required
          />
        </label>
        <label className="space-y-2">
          <span className="text-xs font-black uppercase tracking-wide text-slate-500">Reason</span>
          <select
            value={welfareReason}
            onChange={(event) => setWelfareReason(event.target.value as WelfareCheckReason)}
            className="h-10 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
          >
            {welfareReasons.map((reason) => (
              <option key={reason} value={reason}>
                {welfareReasonLabels[reason]}
              </option>
            ))}
          </select>
        </label>
        <label className="space-y-2 md:col-span-2">
          <span className="text-xs font-black uppercase tracking-wide text-slate-500">Detail (optional)</span>
          <textarea
            value={welfareDetail}
            maxLength={1000}
            onChange={(event) => setWelfareDetail(event.target.value)}
            placeholder="Anything Zorba should know before calling?"
            rows={3}
            className="w-full rounded-xl border border-slate-200 bg-white px-3 py-2 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
          />
          <p className="text-xs font-semibold text-slate-400">{welfareDetail.length}/1000</p>
        </label>
        <div className="flex justify-end md:col-span-2">
          <Button type="submit" variant="healthcare" disabled={creatingWelfareCheck}>
            <HeartPulse className="h-4 w-4" />
            {creatingWelfareCheck ? "Scheduling..." : "Schedule welfare check"}
          </Button>
        </div>
      </form>

      <div className="mt-6 grid gap-4">
        {welfareChecks.map((check) => {
          const status = check.latest_run_status || check.status || "scheduled";
          const cancellable = ["scheduled", "pending"].includes((check.status || "").toLowerCase());

          return (
            <article key={check.id} className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
                <div className="min-w-0">
                  <h3 className="text-base font-black text-slate-950">
                    {check.reason_code
                      ? welfareReasonLabels[check.reason_code as WelfareCheckReason] ??
                        check.reason_code.replaceAll("_", " ")
                      : "Welfare check"}
                  </h3>
                  <p className="mt-2 text-sm font-semibold text-slate-500">
                    {formatDateTime(check.scheduled_at, "Time not set")}
                  </p>
                  {check.reason_detail ? (
                    <p className="mt-3 text-sm leading-6 text-slate-600">{check.reason_detail}</p>
                  ) : null}
                  {check.latest_run_failure_reason ? (
                    <p className="mt-3 text-sm font-semibold text-rose-700">
                      {check.latest_run_failure_reason}
                    </p>
                  ) : null}
                  {typeof check.latest_run_attempts === "number" && check.latest_run_attempts > 0 ? (
                    <p className="mt-1 text-xs font-semibold text-slate-400">
                      Attempts: {check.latest_run_attempts}
                    </p>
                  ) : null}
                </div>
                <div className="flex shrink-0 items-center gap-2">
                  <span className="rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-black uppercase text-emerald-700">
                    {status.replaceAll("_", " ")}
                  </span>
                  {cancellable ? (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      disabled={cancellingWelfareCheck === check.id}
                      onClick={() => onCancel(check.id)}
                    >
                      {cancellingWelfareCheck === check.id ? "Cancelling..." : "Cancel"}
                    </Button>
                  ) : null}
                </div>
              </div>
            </article>
          );
        })}
        {welfareChecks.length === 0 ? (
          <div className="rounded-2xl border border-dashed border-slate-200 bg-slate-50 px-4 py-12 text-center text-sm font-semibold text-slate-400">
            No welfare checks scheduled yet.
          </div>
        ) : null}
      </div>
    </section>
  );
}

function PatientMeetingCard({
  meeting,
  mutating,
  onCancel,
}: {
  meeting: HospitalMeetingRecord;
  mutating: boolean;
  onCancel: () => void;
}) {
  const status = meeting.status || "pending";
  // Backend uses "scheduled" after clinician accept (legacy "accepted" kept for older payloads).
  const confirmed = status === "scheduled" || status === "accepted";
  const cancelled = status === "cancelled";

  return (
    <article className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
      <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <span
              className={`inline-flex items-center gap-1 rounded-full px-2.5 py-1 text-xs font-black uppercase ${
                cancelled
                  ? "bg-slate-100 text-slate-500"
                  : confirmed
                    ? "bg-emerald-50 text-emerald-700"
                    : "bg-orange-50 text-orange-700"
              }`}
            >
              {cancelled ? <VideoOff className="h-3.5 w-3.5" /> : <Video className="h-3.5 w-3.5" />}
              {status.replaceAll("_", " ")}
            </span>
            <span className="text-xs font-semibold text-slate-400">
              {meeting.duration_minutes || 30} min
            </span>
          </div>
          <h3 className="mt-3 text-base font-black text-slate-950">
            {meeting.title || "Zorba Health video visit"}
          </h3>
          <div className="mt-3 grid gap-2 text-sm font-semibold text-slate-500 sm:grid-cols-2">
            <p>Staff ID: {meeting.staff_id || "Unassigned"}</p>
            <p>{formatDateTime(meeting.starts_at, "Time not set")}</p>
            <p>Hospital ID: {meeting.hospital_id || "N/A"}</p>
            <p>{meeting.timezone || "UTC"}</p>
          </div>
        </div>
        <div className="flex flex-wrap gap-2 lg:justify-end">
          {meeting.join_url && !cancelled ? (
            <Button type="button" size="sm" variant="outline" asChild>
              <a href={meeting.join_url} target="_blank" rel="noreferrer">
                <Video className="h-4 w-4" />
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
              <VideoOff className="h-4 w-4" />
              Cancel
            </Button>
          ) : null}
        </div>
      </div>
    </article>
  );
}

function AuditTable({
  events,
}: {
  events: AuditEventRecord[];
}) {
  const [typeFilter, setTypeFilter] = useState("all");
  const [startFilter, setStartFilter] = useState("");
  const [endFilter, setEndFilter] = useState("");

  const eventTypes = useMemo(() => getAuditEventTypeOptions(events), [events]);
  const filteredEvents = useMemo(
    () => filterAuditEvents(events, typeFilter, startFilter, endFilter),
    [endFilter, events, startFilter, typeFilter],
  );

  return (
    <section className="clinical-card overflow-hidden">
      <div className="border-b border-slate-200 p-6">
        <h2 className="flex items-center gap-2 text-lg font-black">
          <Shield className="h-5 w-5 text-indigo-600" />
          Audit Trail
        </h2>
        <div className="mt-5 grid gap-3 md:grid-cols-3">
          <div className="space-y-2">
            <label htmlFor="audit-type" className="text-xs font-black uppercase tracking-wide text-slate-500">
              Event Type
            </label>
            <select
              id="audit-type"
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
            <label htmlFor="audit-start" className="text-xs font-black uppercase tracking-wide text-slate-500">
              From
            </label>
            <input
              id="audit-start"
              type="datetime-local"
              value={startFilter}
              onChange={(event) => setStartFilter(event.target.value)}
              className="h-11 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
            />
          </div>
          <div className="space-y-2">
            <label htmlFor="audit-end" className="text-xs font-black uppercase tracking-wide text-slate-500">
              To
            </label>
            <input
              id="audit-end"
              type="datetime-local"
              value={endFilter}
              onChange={(event) => setEndFilter(event.target.value)}
              className="h-11 w-full rounded-xl border border-slate-200 bg-white px-3 text-sm font-semibold text-slate-700 outline-none focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
            />
          </div>
        </div>
      </div>
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
            No audit history matches the current filters.
          </p>
        ) : null}
      </div>
    </section>
  );
}
