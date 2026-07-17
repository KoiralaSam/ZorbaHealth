"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  Activity,
  CalendarClock,
  ClipboardCheck,
  FileQuestion,
  HeartPulse,
  LogOut,
  PhoneCall,
  ShieldCheck,
} from "lucide-react";
import { Button } from "../../components/ui/button";
import {
  APIEndpoints,
  ConsentRecord,
  HTTPPatientAuditResponse,
  HTTPPatientCallListResponse,
  HTTPPatientConsentListResponse,
  HTTPPatientConsentMutationRequest,
  HTTPPatientConsentMutationResponse,
  HTTPPatientHealthAnswerRequest,
  HTTPPatientHealthAnswerResponse,
  HTTPPatientProfileResponse,
  HTTPPatientWelfareCheckCreateRequest,
  HTTPPatientWelfareCheckListResponse,
  HTTPPatientWelfareCheckResponse,
  PatientCallSummary,
  PatientWelfareCheck,
  WelfareCheckReason,
} from "../../contracts";
import { API_URL } from "../../constants";
import { usePatientLocationSession } from "../../hooks/usePatientLocationSession";
import { cachedJSON, clearClientCache, preloadJSON } from "../../lib/client-cache";

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

export default function PatientHomePage() {
  const router = useRouter();
  /** null = not yet read from sessionStorage (avoid redirect race after login) */
  const [accessToken, setAccessToken] = useState<string | null>(null);
  useEffect(() => {
    setAccessToken(window.sessionStorage.getItem("patient_access_token") ?? "");
  }, []);

  const [profile, setProfile] =
    useState<HTTPPatientProfileResponse["data"] | null>(null);
  const [consents, setConsents] = useState<ConsentRecord[]>([]);
  const [calls, setCalls] = useState<PatientCallSummary[]>([]);
  const [welfareChecks, setWelfareChecks] = useState<PatientWelfareCheck[]>([]);
  const [auditSummary, setAuditSummary] = useState<string[]>([]);
  const [question, setQuestion] = useState("");
  const [answer, setAnswer] = useState("");
  const [answerSources, setAnswerSources] = useState<string[]>([]);
  const [welfareScheduledAt, setWelfareScheduledAt] = useState(localDateTimeValue);
  const [welfareReason, setWelfareReason] =
    useState<WelfareCheckReason>("daily_checkup");
  const [welfareDetail, setWelfareDetail] = useState("");
  const [error, setError] = useState("");
  const [locationNotice, setLocationNotice] = useState("");
  const [loading, setLoading] = useState(true);
  const [mutatingConsent, setMutatingConsent] = useState<string | null>(null);
  const [askingQuestion, setAskingQuestion] = useState(false);
  const [creatingWelfareCheck, setCreatingWelfareCheck] = useState(false);
  const [cancellingWelfareCheck, setCancellingWelfareCheck] = useState<string | null>(null);

  useEffect(() => {
    if (accessToken === null) {
      return;
    }
    if (!accessToken) {
      setLoading(false);
      router.replace("/login/patient");
      return;
    }

    const loadDashboard = async () => {
      setLoading(true);
      setError("");
      try {
        const [profileData, consentsData] = await Promise.all([
          cachedJSON<HTTPPatientProfileResponse>(
            `${API_URL}${APIEndpoints.PATIENT_PROFILE}`,
            accessToken,
            { ttlMs: 30_000 },
          ),
          cachedJSON<HTTPPatientConsentListResponse>(
            `${API_URL}${APIEndpoints.PATIENT_CONSENTS}`,
            accessToken,
            { ttlMs: 30_000 },
          ),
        ]);

        setProfile(profileData.data ?? null);
        setConsents(consentsData.data?.consents ?? []);
        setLoading(false);

        void (async () => {
          try {
            const [callsData, welfareData, auditData] = await Promise.all([
              cachedJSON<HTTPPatientCallListResponse>(
                `${API_URL}${APIEndpoints.PATIENT_CALLS}`,
                accessToken,
                { ttlMs: 30_000 },
              ),
              cachedJSON<HTTPPatientWelfareCheckListResponse>(
                `${API_URL}${APIEndpoints.PATIENT_WELFARE_CHECKS}`,
                accessToken,
                { ttlMs: 30_000 },
              ),
              cachedJSON<HTTPPatientAuditResponse>(
                `${API_URL}${APIEndpoints.PATIENT_AUDIT}`,
                accessToken,
                { ttlMs: 30_000 },
              ),
            ]);
            setCalls(callsData.data?.calls ?? []);
            setWelfareChecks(welfareData.data?.welfare_checks ?? []);
            setAuditSummary(
              (auditData.data?.events ?? []).slice(0, 5).map((event) => {
                const when = event.timestamp
                  ? new Date(event.timestamp).toLocaleString()
                  : "Recently";
                return `${when}: ${event.event_type?.replaceAll("_", " ") ?? "Event"}`;
              }),
            );
          } catch {
            // Keep the core profile/consent page usable if secondary panels fail.
          }
        })();

        preloadJSON<HTTPPatientCallListResponse>(
          `${API_URL}${APIEndpoints.PATIENT_CALLS}`,
          accessToken,
          { ttlMs: 30_000 },
        );
        preloadJSON<HTTPPatientWelfareCheckListResponse>(
          `${API_URL}${APIEndpoints.PATIENT_WELFARE_CHECKS}`,
          accessToken,
          { ttlMs: 30_000 },
        );
        preloadJSON<HTTPPatientAuditResponse>(
          `${API_URL}${APIEndpoints.PATIENT_AUDIT}`,
          accessToken,
          { ttlMs: 30_000 },
        );
      } catch {
        setError("Network error. Please refresh and try again.");
      } finally {
        setLoading(false);
      }
    };

    void loadDashboard();
  }, [accessToken, router]);

  const { locationPermissionBlocked, retryBrowserLocation } =
    usePatientLocationSession(
      accessToken,
      consents,
      setConsents,
      setLocationNotice,
    );

  const consentState = useMemo(() => {
    const map = new Map<string, ConsentRecord>();
    for (const consent of consents) {
      if (consent.consent_type && !map.has(consent.consent_type)) {
        map.set(consent.consent_type, consent);
      }
    }
    return map;
  }, [consents]);

  const mutateConsent = async (consentType: string, enabled: boolean) => {
    if (!accessToken) {
      return;
    }
    setMutatingConsent(consentType);
    setError("");
    const payload: HTTPPatientConsentMutationRequest = {
      consent_type: consentType,
      source: "patient-web-portal",
    };

    try {
      const response = await fetch(`${API_URL}${APIEndpoints.PATIENT_CONSENTS}`, {
        method: enabled ? "POST" : "DELETE",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${accessToken}`,
        },
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
        if (data.data?.consent) {
          next.unshift(data.data.consent);
        }
        return next;
      });
      clearClientCache();
    } catch {
      setError("Network error while updating consent.");
    } finally {
      setMutatingConsent(null);
    }
  };

  const askHealthQuestion = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!accessToken || !question.trim()) {
      return;
    }

    setAskingQuestion(true);
    setError("");
    setAnswer("");
    setAnswerSources([]);

    const payload: HTTPPatientHealthAnswerRequest = {
      question: question.trim(),
      top_k: 5,
    };

    try {
      const response = await fetch(
        `${API_URL}${APIEndpoints.PATIENT_RECORDS_ANSWER}`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${accessToken}`,
          },
          body: JSON.stringify(payload),
        },
      );
      const data: HTTPPatientHealthAnswerResponse = await response.json();
      if (!response.ok) {
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

  const createWelfareCheck = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!accessToken || creatingWelfareCheck) {
      return;
    }

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
    const payload: HTTPPatientWelfareCheckCreateRequest = {
      scheduled_at: scheduledAt.toISOString(),
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC",
      reason_code: welfareReason,
      reason_detail: welfareDetail.trim(),
    };

    try {
      const response = await fetch(
        `${API_URL}${APIEndpoints.PATIENT_WELFARE_CHECKS}`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${accessToken}`,
          },
          body: JSON.stringify(payload),
        },
      );
      const data: HTTPPatientWelfareCheckResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to schedule welfare check.");
        return;
      }
      if (data.data?.welfare_check) {
        setWelfareChecks((current) => [data.data!.welfare_check!, ...current]);
      }
      clearClientCache();
      setWelfareDetail("");
      setWelfareScheduledAt(localDateTimeValue());
    } catch {
      setError("Network error while scheduling welfare check.");
    } finally {
      setCreatingWelfareCheck(false);
    }
  };

  const cancelWelfareCheck = async (id?: string) => {
    if (!accessToken || !id) {
      return;
    }
    setCancellingWelfareCheck(id);
    setError("");
    try {
      const response = await fetch(
        `${API_URL}${APIEndpoints.PATIENT_WELFARE_CHECKS}/${encodeURIComponent(id)}`,
        {
          method: "DELETE",
          headers: { Authorization: `Bearer ${accessToken}` },
        },
      );
      const data: HTTPPatientWelfareCheckResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Unable to cancel welfare check.");
        return;
      }
      if (data.data?.welfare_check) {
        setWelfareChecks((current) =>
          current.map((item) =>
            item.id === id ? data.data!.welfare_check! : item,
          ),
        );
      }
      clearClientCache();
    } catch {
      setError("Network error while cancelling welfare check.");
    } finally {
      setCancellingWelfareCheck(null);
    }
  };

  if (loading) {
    return (
      <main className="min-h-screen bg-[#f5faf8] px-4 py-10">
        <div className="mx-auto max-w-6xl rounded-lg border border-slate-200 bg-white px-5 py-4 text-sm text-slate-600">
          Loading your patient home...
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-[#f5faf8] text-slate-950">
      <div className="flex min-h-screen flex-col">
        <header className="w-full border-b border-slate-200 bg-white">
          <div className="mx-auto flex max-w-6xl items-center justify-between px-4 py-4">
            <button
              type="button"
              onClick={() => router.push("/")}
              className="flex items-center gap-3"
            >
              <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-teal-700 text-white">
                <HeartPulse className="h-5 w-5" aria-hidden="true" />
              </div>
              <div className="text-left">
                <span className="block text-base font-semibold text-slate-950">
                  Zorba Health
                </span>
                <span className="block text-xs text-slate-500">Patient portal</span>
              </div>
            </button>

            <button
              type="button"
              className="inline-flex items-center gap-2 rounded-md border border-slate-200 px-3 py-2 text-sm font-medium text-slate-600 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-950"
              onClick={() => {
                clearClientCache();
                window.sessionStorage.removeItem("patient_access_token");
                window.sessionStorage.removeItem("patient_id");
                router.push("/login/patient");
              }}
            >
              <LogOut className="h-4 w-4" aria-hidden="true" />
              Sign out
            </button>
          </div>
        </header>

        <div className="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-5 px-4 py-6 md:py-8">
          <section className="rounded-lg border border-teal-100 bg-white p-5 md:p-6">
            <p className="text-sm font-semibold uppercase text-teal-700">
              Patient home
            </p>
            <h1 className="mt-2 text-3xl font-semibold tracking-tight text-slate-950">
              {profile?.full_name || "Welcome back"}
            </h1>
            <p className="mt-3 max-w-3xl text-sm leading-6 text-slate-600">
              Manage consent, review recent call summaries, and ask a
              record-grounded question before your next voice session.
            </p>
          </section>

          {error ? (
            <div className="rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {error}
            </div>
          ) : null}

          {locationNotice || locationPermissionBlocked ? (
            <div className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
              <p>{locationNotice}</p>
              {locationPermissionBlocked ? (
                <Button
                  type="button"
                  className="mt-3 bg-amber-800 text-white hover:bg-amber-900"
                  onClick={() => retryBrowserLocation()}
                >
                  Try again
                </Button>
              ) : null}
            </div>
          ) : null}

          <div className="grid gap-6 md:grid-cols-[1.2fr_0.8fr]">
            <section className="rounded-lg border border-slate-200 bg-white p-5">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-slate-950">
                <ShieldCheck className="h-5 w-5 text-teal-700" aria-hidden="true" />
                Profile and voice access
              </h2>
              <div className="mt-4 grid gap-3 text-sm text-slate-700 sm:grid-cols-2">
                <div>
                  <p className="font-medium text-slate-950">Phone</p>
                  <p>{profile?.phone_number || "Not available"}</p>
                </div>
                <div>
                  <p className="font-medium text-slate-950">Email</p>
                  <p>{profile?.email || "Not available"}</p>
                </div>
                <div>
                  <p className="font-medium text-slate-950">Date of birth</p>
                  <p>{profile?.date_of_birth || "Not available"}</p>
                </div>
                <div>
                  <p className="font-medium text-slate-950">Support window</p>
                  <p>{profile?.support_window || "24/7"}</p>
                </div>
              </div>
              <div className="mt-5 rounded-lg border border-teal-100 bg-teal-50 p-4">
                <p className="text-sm font-semibold text-teal-950">
                  Start a voice session
                </p>
                <p className="mt-1 text-sm leading-6 text-teal-900">
                  Call Zorba directly for medication, follow-up, or emergency
                  support.
                </p>
                <div className="mt-4 flex flex-col gap-3 sm:flex-row">
                  <a
                    href={`tel:${profile?.voice_phone || "+1-800-ZORBA-AI"}`}
                    className="inline-flex items-center justify-center gap-2 rounded-md bg-teal-700 px-4 py-3 text-sm font-semibold text-white hover:bg-teal-800"
                  >
                    <PhoneCall className="h-4 w-4" aria-hidden="true" />
                    Call {profile?.voice_phone || "Zorba"}
                  </a>
                  <p className="self-center text-xs text-teal-900">
                    Voice assistant enabled: {profile?.voice_enabled ? "Yes" : "No"}
                  </p>
                </div>
              </div>
            </section>

            <section className="rounded-lg border border-slate-200 bg-white p-5">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-slate-950">
                <Activity className="h-5 w-5 text-blue-700" aria-hidden="true" />
                Recent activity
              </h2>
              <div className="mt-4 space-y-3 text-sm text-slate-700">
                {auditSummary.length > 0 ? (
                  auditSummary.map((item) => (
                    <div key={item} className="rounded-md border border-slate-100 bg-slate-50 px-3 py-2">
                      {item}
                    </div>
                  ))
                ) : (
                  <p>No audit events available yet.</p>
                )}
              </div>
            </section>
          </div>

          <section className="rounded-lg border border-slate-200 bg-white p-5">
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h2 className="flex items-center gap-2 text-lg font-semibold text-slate-950">
                  <CalendarClock className="h-5 w-5 text-teal-700" aria-hidden="true" />
                  Scheduled welfare checks
                </h2>
                <p className="mt-1 text-sm text-slate-600">
                  Schedule a phone check-in from Zorba using your saved profile number.
                </p>
              </div>
            </div>
            <form
              onSubmit={createWelfareCheck}
              className="mt-5 grid gap-4 lg:grid-cols-[1fr_1fr]">
              <div>
                <label className="text-sm font-medium text-slate-950" htmlFor="welfare-time">
                  Date and time
                </label>
                <input
                  id="welfare-time"
                  type="datetime-local"
                  value={welfareScheduledAt}
                  onChange={(e) => setWelfareScheduledAt(e.target.value)}
                  className="mt-2 w-full rounded-md border border-slate-300 bg-white px-4 py-3 text-sm focus:border-teal-500 focus:outline-none focus:ring-2 focus:ring-teal-100"
                />
              </div>
              <div>
                <label className="text-sm font-medium text-slate-950" htmlFor="welfare-reason">
                  Reason
                </label>
                <select
                  id="welfare-reason"
                  value={welfareReason}
                  onChange={(e) => setWelfareReason(e.target.value as WelfareCheckReason)}
                  className="mt-2 w-full rounded-md border border-slate-300 bg-white px-4 py-3 text-sm focus:border-teal-500 focus:outline-none focus:ring-2 focus:ring-teal-100"
                >
                  {welfareReasons.map((reason) => (
                    <option key={reason} value={reason}>
                      {welfareReasonLabels[reason]}
                    </option>
                  ))}
                </select>
              </div>
              <div className="lg:col-span-2">
                <label className="text-sm font-medium text-slate-950" htmlFor="welfare-detail">
                  Detail
                </label>
                <textarea
                  id="welfare-detail"
                  value={welfareDetail}
                  maxLength={1000}
                  onChange={(e) => setWelfareDetail(e.target.value)}
                  placeholder="Anything Zorba should know before calling?"
                  className="mt-2 min-h-24 w-full rounded-md border border-slate-300 bg-white px-4 py-3 text-sm focus:border-teal-500 focus:outline-none focus:ring-2 focus:ring-teal-100"
                />
                <p className="mt-1 text-xs text-slate-500">
                  {welfareDetail.length}/1000
                </p>
              </div>
              <div className="lg:col-span-2">
                <Button
                  type="submit"
                  className="bg-teal-700 text-white hover:bg-teal-800"
                  disabled={creatingWelfareCheck}
                >
                  {creatingWelfareCheck ? "Scheduling..." : "Schedule welfare check"}
                </Button>
              </div>
            </form>

            <div className="mt-6 space-y-3">
              {welfareChecks.length > 0 ? (
                welfareChecks.map((check) => {
                  const scheduled = check.scheduled_at
                    ? new Date(check.scheduled_at).toLocaleString()
                    : "Not scheduled";
                  const cancellable = ["scheduled", "pending"].includes(
                    (check.status || "").toLowerCase(),
                  );
                  return (
                    <div key={check.id} className="rounded-lg border border-slate-200 p-4">
                      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                        <div>
                          <p className="text-sm font-semibold text-slate-950">
                            {check.reason_code
                              ? welfareReasonLabels[check.reason_code as WelfareCheckReason] ??
                                check.reason_code.replaceAll("_", " ")
                              : "Welfare check"}
                          </p>
                          <p className="mt-1 text-xs text-slate-500">{scheduled}</p>
                          {check.reason_detail ? (
                            <p className="mt-2 text-sm text-slate-700">
                              {check.reason_detail}
                            </p>
                          ) : null}
                          {check.latest_run_failure_reason ? (
                            <p className="mt-2 text-xs text-rose-700">
                              {check.latest_run_failure_reason}
                            </p>
                          ) : null}
                          {typeof check.latest_run_attempts === "number" &&
                          check.latest_run_attempts > 0 ? (
                            <p className="mt-1 text-xs text-slate-500">
                              Attempts: {check.latest_run_attempts}
                            </p>
                          ) : null}
                        </div>
                        <div className="flex items-center gap-2">
                          <span className="rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-semibold text-emerald-700">
                            {check.latest_run_status || check.status || "scheduled"}
                          </span>
                          {cancellable ? (
                            <Button
                              type="button"
                              className="bg-slate-900 text-white hover:bg-slate-800"
                              disabled={cancellingWelfareCheck === check.id}
                              onClick={() => cancelWelfareCheck(check.id)}
                            >
                              {cancellingWelfareCheck === check.id ? "Cancelling..." : "Cancel"}
                            </Button>
                          ) : null}
                        </div>
                      </div>
                    </div>
                  );
                })
              ) : (
                <div className="rounded-lg border border-slate-100 bg-slate-50 px-4 py-3 text-sm text-slate-600">
                  No welfare checks scheduled yet.
                </div>
              )}
            </div>
          </section>

          <section className="rounded-lg border border-slate-200 bg-white p-5">
            <h2 className="flex items-center gap-2 text-lg font-semibold text-slate-950">
              <ClipboardCheck className="h-5 w-5 text-teal-700" aria-hidden="true" />
              Consent center
            </h2>
            <p className="mt-2 text-sm leading-6 text-slate-600">
              Sensitive tools only run after explicit consent. You can review
              and update the current state here.
            </p>
            <div className="mt-4 grid gap-3 md:grid-cols-2">
              {consentTypes.map((consentType) => {
                const current = consentState.get(consentType);
                const active = current?.status === "active";
                return (
                  <div
                    key={consentType}
                    className="rounded-lg border border-slate-200 p-4"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <p className="font-semibold text-slate-950">
                          {consentLabels[consentType]}
                        </p>
                        <p className="mt-1 text-xs text-slate-500">
                          Status: {current?.status || "not granted"}
                        </p>
                      </div>
                      <Button
                        type="button"
                        className={
                          active
                            ? "bg-slate-900 text-white hover:bg-slate-800"
                            : "bg-teal-700 text-white hover:bg-teal-800"
                        }
                        disabled={mutatingConsent === consentType}
                        onClick={() =>
                          mutateConsent(consentType, !active)
                        }
                      >
                        {mutatingConsent === consentType
                          ? "Saving..."
                          : active
                            ? "Revoke"
                            : "Grant"}
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          </section>

          <div className="grid gap-6 md:grid-cols-[1fr_1fr]">
            <section className="rounded-lg border border-slate-200 bg-white p-5">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-slate-950">
                <FileQuestion className="h-5 w-5 text-blue-700" aria-hidden="true" />
                Ask about your records
              </h2>
              <p className="mt-2 text-sm leading-6 text-slate-600">
                This uses the patient records service with the same patient
                access checks as the voice assistant.
              </p>
              <form onSubmit={askHealthQuestion} className="mt-4 space-y-4">
                <textarea
                  value={question}
                  onChange={(e) => setQuestion(e.target.value)}
                  placeholder="Example: What were my last recorded allergies or medication changes?"
                  className="min-h-28 w-full rounded-md border border-slate-300 bg-white px-4 py-3 text-sm focus:border-teal-500 focus:outline-none focus:ring-2 focus:ring-teal-100"
                />
                <Button
                  type="submit"
                  className="bg-teal-700 text-white hover:bg-teal-800"
                  disabled={askingQuestion}
                >
                  {askingQuestion ? "Answering..." : "Ask question"}
                </Button>
              </form>
              <div className="mt-4 rounded-lg border border-slate-100 bg-slate-50 p-4">
                <p className="text-sm font-semibold text-slate-950">Answer</p>
                <p className="mt-2 whitespace-pre-wrap text-sm leading-6 text-slate-700">
                  {answer || "No answer yet."}
                </p>
                {answerSources.length > 0 ? (
                  <div className="mt-3">
                    <p className="text-xs font-semibold uppercase text-slate-500">
                      Sources
                    </p>
                    <ul className="mt-2 space-y-1 text-xs text-slate-600">
                      {answerSources.map((source) => (
                        <li key={source}>{source}</li>
                      ))}
                    </ul>
                  </div>
                ) : null}
              </div>
            </section>

            <section className="rounded-lg border border-slate-200 bg-white p-5">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-slate-950">
                <PhoneCall className="h-5 w-5 text-teal-700" aria-hidden="true" />
                Recent call summaries
              </h2>
              <p className="mt-2 text-sm leading-6 text-slate-600">
                Stored summaries are shown here without exposing full raw
                transcripts by default.
              </p>
              <div className="mt-4 space-y-3">
                {calls.length > 0 ? (
                  calls.map((call) => (
                    <div key={call.id} className="rounded-lg border border-slate-200 p-4">
                      <div className="flex items-center justify-between gap-3">
                        <p className="text-sm font-semibold text-slate-950">
                          Call #{call.id}
                        </p>
                        <span className="rounded-full bg-blue-50 px-2.5 py-1 text-xs font-semibold text-blue-700">
                          {call.status || "unknown"}
                        </span>
                      </div>
                      <p className="mt-2 text-xs text-slate-500">
                        Started:{" "}
                        {call.started_at
                          ? new Date(call.started_at).toLocaleString()
                          : "Unavailable"}
                      </p>
                      <p className="mt-3 text-sm leading-6 text-slate-700">
                        {call.summary || "No summary available yet."}
                      </p>
                    </div>
                  ))
                ) : (
                  <div className="rounded-lg border border-slate-100 bg-slate-50 px-4 py-3 text-sm text-slate-600">
                    No call summaries have been captured yet.
                  </div>
                )}
              </div>
            </section>
          </div>
        </div>
      </div>
    </main>
  );
}
