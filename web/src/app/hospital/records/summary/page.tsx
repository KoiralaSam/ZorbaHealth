"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  ClipboardList,
  HeartPulse,
  LogOut,
  Search,
  ShieldAlert,
  Stethoscope,
} from "lucide-react";
import { Button } from "../../../../components/ui/button";
import {
  APIEndpoints,
  AuditEventRecord,
  HospitalIncidentRecord,
  HTTPHospitalIncidentListResponse,
  HTTPHospitalPatientAuditResponse,
  HTTPHospitalPatientSummaryRequest,
  HTTPHospitalPatientSummaryResponse,
} from "../../../../contracts";
import { API_URL } from "../../../../constants";
import { cachedJSON, clearClientCache, preloadJSON } from "../../../../lib/client-cache";

const focusOptions = [
  { value: "full", label: "Full summary" },
  { value: "medications", label: "Medications" },
  { value: "allergies", label: "Allergies" },
  { value: "diagnoses", label: "Diagnoses" },
];

export default function HospitalPatientSummaryPage() {
  const router = useRouter();
  const accessToken = useMemo(() => {
    if (typeof window === "undefined") {
      return "";
    }
    return window.sessionStorage.getItem("hospital_access_token") ?? "";
  }, []);

  const [patientID, setPatientID] = useState("");
  const [focus, setFocus] = useState("full");
  const [summary, setSummary] = useState("");
  const [error, setError] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [incidents, setIncidents] = useState<HospitalIncidentRecord[]>([]);
  const [auditEvents, setAuditEvents] = useState<AuditEventRecord[]>([]);

  const loadIncidents = useCallback(async () => {
    if (!accessToken) {
      return;
    }
    try {
      const data = await cachedJSON<HTTPHospitalIncidentListResponse>(
        `${API_URL}${APIEndpoints.HOSPITAL_INCIDENTS}`,
        accessToken,
        { ttlMs: 30_000 },
      );
      setIncidents(data.data?.incidents ?? []);
    } catch {
      // best-effort side panel
    }
  }, [accessToken]);

  useEffect(() => {
    void loadIncidents();
    if (accessToken) {
      preloadJSON<HTTPHospitalIncidentListResponse>(
        `${API_URL}${APIEndpoints.HOSPITAL_INCIDENTS}`,
        accessToken,
        { ttlMs: 30_000 },
      );
    }
  }, [accessToken, loadIncidents]);

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
      const response = await fetch(`${API_URL}${APIEndpoints.HOSPITAL_PATIENT_SUMMARY}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${accessToken}`,
        },
        body: JSON.stringify(payload),
      });
      const data: HTTPHospitalPatientSummaryResponse = await response.json();

      if (!response.ok) {
        setError(data.error?.message || "Failed to summarize patient records.");
        return;
      }

      setSummary(data.data?.summary || "No summary was returned.");
      const auditData = await cachedJSON<HTTPHospitalPatientAuditResponse>(
        `${API_URL}${APIEndpoints.HOSPITAL_PATIENT_AUDIT}?patient_id=${encodeURIComponent(payload.patient_id)}`,
        accessToken,
        { ttlMs: 30_000, force: true },
      );
      setAuditEvents(auditData.data?.events ?? []);
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setIsLoading(false);
    }
  };

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
                <span className="block text-base font-semibold text-slate-950">Zorba Health</span>
                <span className="block text-xs text-slate-500">Hospital console</span>
              </div>
            </button>

            <button
              type="button"
              onClick={() => {
                clearClientCache();
                window.sessionStorage.removeItem("hospital_access_token");
                router.push("/login/hospital");
              }}
              className="inline-flex items-center gap-2 rounded-md border border-slate-200 px-3 py-2 text-sm font-medium text-slate-600 hover:border-slate-300 hover:bg-slate-50 hover:text-slate-950"
            >
              <LogOut className="h-4 w-4" aria-hidden="true" />
              Sign out
            </button>
          </div>
        </header>

        <div className="mx-auto flex w-full max-w-6xl flex-1 flex-col gap-5 px-4 py-6 md:py-8">
          <div className="rounded-lg border border-teal-100 bg-white p-5 md:p-6">
            <p className="text-sm font-semibold uppercase text-teal-700">Clinical records</p>
            <h1 className="mt-2 text-3xl font-semibold tracking-tight text-slate-950">Patient record summary</h1>
            <p className="mt-3 max-w-3xl text-sm leading-6 text-slate-600">
              Enter a patient ID and choose a focus area to generate a quick staff-facing summary.
            </p>
          </div>

          <form onSubmit={handleSubmit} className="rounded-lg border border-slate-200 bg-white p-5">
            <div className="grid gap-4 md:grid-cols-[2fr_1fr_auto]">
              <div>
                <label htmlFor="patientID" className="mb-2 block text-sm font-medium text-slate-950">
                  Patient ID
                </label>
                <input
                  id="patientID"
                  value={patientID}
                  onChange={(e) => setPatientID(e.target.value)}
                  placeholder="e.g. 5e2e6f0a-..."
                  className="w-full rounded-md border border-slate-300 bg-white px-4 py-3 text-sm focus:border-teal-500 focus:outline-none focus:ring-2 focus:ring-teal-100"
                  required
                />
              </div>

              <div>
                <label htmlFor="focus" className="mb-2 block text-sm font-medium text-slate-950">
                  Focus
                </label>
                <select
                  id="focus"
                  value={focus}
                  onChange={(e) => setFocus(e.target.value)}
                  className="w-full rounded-md border border-slate-300 bg-white px-4 py-3 text-sm focus:border-teal-500 focus:outline-none focus:ring-2 focus:ring-teal-100"
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
                  className="w-full gap-2 bg-teal-700 py-6 text-white hover:bg-teal-800 md:w-auto"
                  disabled={isLoading}
                >
                  <Search className="h-4 w-4" aria-hidden="true" />
                  {isLoading ? "Generating..." : "Generate summary"}
                </Button>
              </div>
            </div>
          </form>

          {error ? (
            <div className="rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
              {error}
            </div>
          ) : null}

          <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
            <section className="rounded-lg border border-slate-200 bg-white p-5">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-slate-950">
                <Stethoscope className="h-5 w-5 text-teal-700" aria-hidden="true" />
                Summary
              </h2>
              <div className="mt-4 min-h-48 whitespace-pre-wrap rounded-lg border border-slate-100 bg-slate-50 p-4 text-sm leading-6 text-slate-700">
                {summary || "No summary generated yet."}
              </div>

              <div className="mt-8">
                <h3 className="flex items-center gap-2 text-base font-semibold text-slate-950">
                  <ClipboardList className="h-4 w-4 text-blue-700" aria-hidden="true" />
                  Patient audit trail
                </h3>
                <div className="mt-3 space-y-3">
                  {auditEvents.length > 0 ? (
                    auditEvents.map((event) => (
                      <div key={event.event_id} className="rounded-lg border border-slate-200 p-4">
                        <div className="flex items-center justify-between gap-3">
                          <p className="text-sm font-semibold text-slate-950">
                            {event.event_type?.replaceAll("_", " ") || "Audit event"}
                          </p>
                          <span className="text-xs text-slate-500">
                            {event.timestamp
                              ? new Date(event.timestamp).toLocaleString()
                              : "Unknown time"}
                          </span>
                        </div>
                        <p className="mt-2 text-xs text-slate-600">
                          Service: {event.service_name || "Unknown"}
                        </p>
                        {event.failure_reason ? (
                          <p className="mt-2 text-xs text-red-700">
                            Failure: {event.failure_reason}
                          </p>
                        ) : null}
                      </div>
                    ))
                  ) : (
                    <div className="rounded-lg border border-slate-100 bg-slate-50 px-4 py-3 text-sm text-slate-600">
                      Generate a summary for a consented patient to inspect recent audit events.
                    </div>
                  )}
                </div>
              </div>
            </section>

            <aside className="rounded-lg border border-slate-200 bg-white p-5">
              <h2 className="flex items-center gap-2 text-lg font-semibold text-slate-950">
                <ShieldAlert className="h-5 w-5 text-red-700" aria-hidden="true" />
                Emergency inbox
              </h2>
              <p className="mt-2 text-sm leading-6 text-slate-600">
                Recent escalations recorded by the voice safety flow.
              </p>
              <div className="mt-4 space-y-3">
                {incidents.length > 0 ? (
                  incidents.map((incident) => (
                    <div key={incident.event_id} className="rounded-lg border border-slate-200 p-4">
                      <div className="flex items-center justify-between gap-3">
                        <p className="text-sm font-semibold text-slate-950">
                          {incident.severity || "unspecified"} severity
                        </p>
                        <span className="text-xs text-slate-500">
                          {incident.timestamp
                            ? new Date(incident.timestamp).toLocaleString()
                            : "Unknown time"}
                        </span>
                      </div>
                      <p className="mt-2 text-xs text-slate-600">
                        Patient: {incident.patient_id || "Not attached"}
                      </p>
                      <p className="mt-1 text-xs text-slate-600">
                        Session: {incident.session_id || "Unknown"}
                      </p>
                    </div>
                  ))
                ) : (
                  <div className="rounded-lg border border-slate-100 bg-slate-50 px-4 py-3 text-sm text-slate-600">
                    No emergency escalations yet.
                  </div>
                )}
              </div>
            </aside>
          </div>
        </div>
      </div>
    </main>
  );
}
