"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
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
      const response = await fetch(`${API_URL}${APIEndpoints.HOSPITAL_INCIDENTS}`, {
        headers: {
          Authorization: `Bearer ${accessToken}`,
        },
      });
      const data: HTTPHospitalIncidentListResponse = await response.json();
      if (response.ok) {
        setIncidents(data.data?.incidents ?? []);
      }
    } catch {
      // best-effort side panel
    }
  }, [accessToken]);

  useEffect(() => {
    void loadIncidents();
  }, [loadIncidents]);

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
      const auditResponse = await fetch(
        `${API_URL}${APIEndpoints.HOSPITAL_PATIENT_AUDIT}?patient_id=${encodeURIComponent(payload.patient_id)}`,
        {
          headers: {
            Authorization: `Bearer ${accessToken}`,
          },
        },
      );
      const auditData: HTTPHospitalPatientAuditResponse =
        await auditResponse.json();
      if (auditResponse.ok) {
        setAuditEvents(auditData.data?.events ?? []);
      } else {
        setAuditEvents([]);
      }
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setIsLoading(false);
    }
  };

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
              <span className="text-lg font-semibold text-gray-900">Zorba Health</span>
            </button>

            <button
              type="button"
              onClick={() => {
                window.sessionStorage.removeItem("hospital_access_token");
                router.push("/login/hospital");
              }}
              className="text-sm text-gray-600 hover:text-gray-900"
            >
              Sign out
            </button>
          </div>
        </header>

        <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col gap-6 px-4 py-8">
          <div>
            <h1 className="text-3xl font-bold text-gray-900">Patient record summary</h1>
            <p className="mt-2 text-sm text-gray-600">
              Enter a patient ID and choose a focus area to generate a quick staff-facing summary.
            </p>
          </div>

          <form onSubmit={handleSubmit} className="rounded-2xl bg-white p-6 shadow-lg">
            <div className="grid gap-4 md:grid-cols-[2fr_1fr_auto]">
              <div>
                <label htmlFor="patientID" className="mb-2 block text-sm font-medium text-gray-700">
                  Patient ID
                </label>
                <input
                  id="patientID"
                  value={patientID}
                  onChange={(e) => setPatientID(e.target.value)}
                  placeholder="e.g. 5e2e6f0a-..."
                  className="w-full rounded-lg border border-gray-300 px-4 py-3 focus:border-transparent focus:ring-2 focus:ring-blue-500"
                  required
                />
              </div>

              <div>
                <label htmlFor="focus" className="mb-2 block text-sm font-medium text-gray-700">
                  Focus
                </label>
                <select
                  id="focus"
                  value={focus}
                  onChange={(e) => setFocus(e.target.value)}
                  className="w-full rounded-lg border border-gray-300 px-4 py-3 focus:border-transparent focus:ring-2 focus:ring-blue-500"
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
                  className="w-full bg-blue-600 py-6 text-white hover:bg-blue-700 md:w-auto"
                  disabled={isLoading}
                >
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
            <section className="rounded-2xl bg-white p-6 shadow-lg">
              <h2 className="text-lg font-semibold text-gray-900">Summary</h2>
              <div className="mt-4 min-h-48 whitespace-pre-wrap text-sm leading-6 text-gray-700">
                {summary || "No summary generated yet."}
              </div>

              <div className="mt-8">
                <h3 className="text-base font-semibold text-gray-900">Patient audit trail</h3>
                <div className="mt-3 space-y-3">
                  {auditEvents.length > 0 ? (
                    auditEvents.map((event) => (
                      <div key={event.event_id} className="rounded-xl border border-gray-200 p-4">
                        <div className="flex items-center justify-between gap-3">
                          <p className="text-sm font-medium text-gray-900">
                            {event.event_type?.replaceAll("_", " ") || "Audit event"}
                          </p>
                          <span className="text-xs text-gray-500">
                            {event.timestamp
                              ? new Date(event.timestamp).toLocaleString()
                              : "Unknown time"}
                          </span>
                        </div>
                        <p className="mt-2 text-xs text-gray-600">
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
                    <div className="rounded-xl bg-gray-50 px-4 py-3 text-sm text-gray-600">
                      Generate a summary for a consented patient to inspect recent audit events.
                    </div>
                  )}
                </div>
              </div>
            </section>

            <aside className="rounded-2xl bg-white p-6 shadow-lg">
              <h2 className="text-lg font-semibold text-gray-900">Emergency inbox</h2>
              <p className="mt-2 text-sm text-gray-600">
                Recent escalations recorded by the voice safety flow.
              </p>
              <div className="mt-4 space-y-3">
                {incidents.length > 0 ? (
                  incidents.map((incident) => (
                    <div key={incident.event_id} className="rounded-xl border border-gray-200 p-4">
                      <div className="flex items-center justify-between gap-3">
                        <p className="text-sm font-medium text-gray-900">
                          {incident.severity || "unspecified"} severity
                        </p>
                        <span className="text-xs text-gray-500">
                          {incident.timestamp
                            ? new Date(incident.timestamp).toLocaleString()
                            : "Unknown time"}
                        </span>
                      </div>
                      <p className="mt-2 text-xs text-gray-600">
                        Patient: {incident.patient_id || "Not attached"}
                      </p>
                      <p className="mt-1 text-xs text-gray-600">
                        Session: {incident.session_id || "Unknown"}
                      </p>
                    </div>
                  ))
                ) : (
                  <div className="rounded-xl bg-gray-50 px-4 py-3 text-sm text-gray-600">
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
