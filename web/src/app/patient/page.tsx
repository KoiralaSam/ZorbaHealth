"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
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
  PatientCallSummary,
} from "../../contracts";
import { API_URL } from "../../constants";
import { usePatientLocationSession } from "../../hooks/usePatientLocationSession";

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
  const [auditSummary, setAuditSummary] = useState<string[]>([]);
  const [question, setQuestion] = useState("");
  const [answer, setAnswer] = useState("");
  const [answerSources, setAnswerSources] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [locationNotice, setLocationNotice] = useState("");
  const [loading, setLoading] = useState(true);
  const [mutatingConsent, setMutatingConsent] = useState<string | null>(null);
  const [askingQuestion, setAskingQuestion] = useState(false);

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
        const headers = {
          Authorization: `Bearer ${accessToken}`,
        };
        const [profileRes, consentsRes, callsRes, auditRes] = await Promise.all([
          fetch(`${API_URL}${APIEndpoints.PATIENT_PROFILE}`, { headers }),
          fetch(`${API_URL}${APIEndpoints.PATIENT_CONSENTS}`, { headers }),
          fetch(`${API_URL}${APIEndpoints.PATIENT_CALLS}`, { headers }),
          fetch(`${API_URL}${APIEndpoints.PATIENT_AUDIT}`, { headers }),
        ]);

        const profileData: HTTPPatientProfileResponse = await profileRes.json();
        const consentsData: HTTPPatientConsentListResponse =
          await consentsRes.json();
        const callsData: HTTPPatientCallListResponse = await callsRes.json();
        const auditData: HTTPPatientAuditResponse = await auditRes.json();

        if (!profileRes.ok) {
          setError(profileData.error?.message || "Failed to load patient home.");
          return;
        }

        setProfile(profileData.data ?? null);
        setConsents(consentsData.data?.consents ?? []);
        setCalls(callsData.data?.calls ?? []);
        setAuditSummary(
          (auditData.data?.events ?? []).slice(0, 5).map((event) => {
            const when = event.timestamp
              ? new Date(event.timestamp).toLocaleString()
              : "Recently";
            return `${when}: ${event.event_type?.replaceAll("_", " ") ?? "Event"}`;
          }),
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

  if (loading) {
    return (
      <main className="min-h-screen bg-gradient-to-b from-blue-50 to-white px-4 py-10">
        <div className="mx-auto max-w-5xl text-sm text-gray-600">
          Loading your patient home...
        </div>
      </main>
    );
  }

  return (
    <main className="min-h-screen bg-gradient-to-b from-blue-50 to-white">
      <div className="flex min-h-screen flex-col">
        <header className="w-full border-b border-blue-100 bg-white/70 backdrop-blur">
          <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-4">
            <button
              type="button"
              onClick={() => router.push("/")}
              className="flex items-center gap-2"
            >
              <div className="flex h-9 w-9 items-center justify-center rounded-full bg-blue-100">
                <span className="text-base font-semibold text-blue-700">Z</span>
              </div>
              <span className="text-lg font-semibold text-gray-900">
                Zorba Health
              </span>
            </button>

            <button
              type="button"
              className="text-sm text-gray-600 hover:text-gray-900"
              onClick={() => {
                window.sessionStorage.removeItem("patient_access_token");
                window.sessionStorage.removeItem("patient_id");
                router.push("/login/patient");
              }}
            >
              Sign out
            </button>
          </div>
        </header>

        <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-6 px-4 py-8">
          <section className="rounded-2xl bg-white p-6 shadow-lg">
            <p className="text-sm font-medium uppercase tracking-wide text-blue-600">
              Patient home
            </p>
            <h1 className="mt-2 text-3xl font-bold text-gray-900">
              {profile?.full_name || "Welcome back"}
            </h1>
            <p className="mt-3 max-w-3xl text-sm text-gray-600">
              Manage consent, review recent call summaries, and ask a
              record-grounded question before your next voice session.
            </p>
          </section>

          {error ? (
            <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {error}
            </div>
          ) : null}

          {locationNotice || locationPermissionBlocked ? (
            <div className="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900">
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
            <section className="rounded-2xl bg-white p-6 shadow-lg">
              <h2 className="text-lg font-semibold text-gray-900">
                Profile and voice access
              </h2>
              <div className="mt-4 grid gap-3 text-sm text-gray-700 sm:grid-cols-2">
                <div>
                  <p className="font-medium text-gray-900">Phone</p>
                  <p>{profile?.phone_number || "Not available"}</p>
                </div>
                <div>
                  <p className="font-medium text-gray-900">Email</p>
                  <p>{profile?.email || "Not available"}</p>
                </div>
                <div>
                  <p className="font-medium text-gray-900">Date of birth</p>
                  <p>{profile?.date_of_birth || "Not available"}</p>
                </div>
                <div>
                  <p className="font-medium text-gray-900">Support window</p>
                  <p>{profile?.support_window || "24/7"}</p>
                </div>
              </div>
              <div className="mt-5 rounded-xl border border-blue-100 bg-blue-50 p-4">
                <p className="text-sm font-medium text-blue-900">
                  Start a voice session
                </p>
                <p className="mt-1 text-sm text-blue-800">
                  Call Zorba directly for medication, follow-up, or emergency
                  support.
                </p>
                <div className="mt-4 flex flex-col gap-3 sm:flex-row">
                  <a
                    href={`tel:${profile?.voice_phone || "+1-800-ZORBA-AI"}`}
                    className="inline-flex items-center justify-center rounded-md bg-blue-600 px-4 py-3 text-sm font-medium text-white hover:bg-blue-700"
                  >
                    Call {profile?.voice_phone || "Zorba"}
                  </a>
                  <p className="self-center text-xs text-blue-900">
                    Voice assistant enabled: {profile?.voice_enabled ? "Yes" : "No"}
                  </p>
                </div>
              </div>
            </section>

            <section className="rounded-2xl bg-white p-6 shadow-lg">
              <h2 className="text-lg font-semibold text-gray-900">
                Recent activity
              </h2>
              <div className="mt-4 space-y-3 text-sm text-gray-700">
                {auditSummary.length > 0 ? (
                  auditSummary.map((item) => (
                    <div key={item} className="rounded-lg bg-gray-50 px-3 py-2">
                      {item}
                    </div>
                  ))
                ) : (
                  <p>No audit events available yet.</p>
                )}
              </div>
            </section>
          </div>

          <section className="rounded-2xl bg-white p-6 shadow-lg">
            <h2 className="text-lg font-semibold text-gray-900">Consent center</h2>
            <p className="mt-2 text-sm text-gray-600">
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
                    className="rounded-xl border border-gray-200 p-4"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <p className="font-medium text-gray-900">
                          {consentLabels[consentType]}
                        </p>
                        <p className="mt-1 text-xs text-gray-500">
                          Status: {current?.status || "not granted"}
                        </p>
                      </div>
                      <Button
                        type="button"
                        className={
                          active
                            ? "bg-gray-900 text-white hover:bg-gray-800"
                            : "bg-blue-600 text-white hover:bg-blue-700"
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
            <section className="rounded-2xl bg-white p-6 shadow-lg">
              <h2 className="text-lg font-semibold text-gray-900">
                Ask about your records
              </h2>
              <p className="mt-2 text-sm text-gray-600">
                This uses the patient records service with the same patient
                access checks as the voice assistant.
              </p>
              <form onSubmit={askHealthQuestion} className="mt-4 space-y-4">
                <textarea
                  value={question}
                  onChange={(e) => setQuestion(e.target.value)}
                  placeholder="Example: What were my last recorded allergies or medication changes?"
                  className="min-h-28 w-full rounded-lg border border-gray-300 px-4 py-3 text-sm focus:border-transparent focus:ring-2 focus:ring-blue-500"
                />
                <Button
                  type="submit"
                  className="bg-blue-600 text-white hover:bg-blue-700"
                  disabled={askingQuestion}
                >
                  {askingQuestion ? "Answering..." : "Ask question"}
                </Button>
              </form>
              <div className="mt-4 rounded-xl bg-gray-50 p-4">
                <p className="text-sm font-medium text-gray-900">Answer</p>
                <p className="mt-2 whitespace-pre-wrap text-sm text-gray-700">
                  {answer || "No answer yet."}
                </p>
                {answerSources.length > 0 ? (
                  <div className="mt-3">
                    <p className="text-xs font-medium uppercase tracking-wide text-gray-500">
                      Sources
                    </p>
                    <ul className="mt-2 space-y-1 text-xs text-gray-600">
                      {answerSources.map((source) => (
                        <li key={source}>{source}</li>
                      ))}
                    </ul>
                  </div>
                ) : null}
              </div>
            </section>

            <section className="rounded-2xl bg-white p-6 shadow-lg">
              <h2 className="text-lg font-semibold text-gray-900">
                Recent call summaries
              </h2>
              <p className="mt-2 text-sm text-gray-600">
                Stored summaries are shown here without exposing full raw
                transcripts by default.
              </p>
              <div className="mt-4 space-y-3">
                {calls.length > 0 ? (
                  calls.map((call) => (
                    <div key={call.id} className="rounded-xl border border-gray-200 p-4">
                      <div className="flex items-center justify-between gap-3">
                        <p className="text-sm font-medium text-gray-900">
                          Call #{call.id}
                        </p>
                        <span className="rounded-full bg-blue-50 px-2.5 py-1 text-xs font-medium text-blue-700">
                          {call.status || "unknown"}
                        </span>
                      </div>
                      <p className="mt-2 text-xs text-gray-500">
                        Started:{" "}
                        {call.started_at
                          ? new Date(call.started_at).toLocaleString()
                          : "Unavailable"}
                      </p>
                      <p className="mt-3 text-sm text-gray-700">
                        {call.summary || "No summary available yet."}
                      </p>
                    </div>
                  ))
                ) : (
                  <div className="rounded-xl bg-gray-50 px-4 py-3 text-sm text-gray-600">
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
