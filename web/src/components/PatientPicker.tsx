"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { apiFetch } from "@/lib/auth-client";
import { APIEndpoints } from "@/contracts";
import type { HospitalPatientRecord } from "@/contracts";
import { fuzzyFilterPatients } from "@/lib/fuzzy-patient";

type Props = {
  value: string;
  onChange: (patientId: string, patient?: HospitalPatientRecord) => void;
  label?: string;
  required?: boolean;
  className?: string;
};

export function PatientPicker({
  value,
  onChange,
  label = "Patient",
  required = false,
  className = "",
}: Props) {
  const rootRef = useRef<HTMLDivElement>(null);
  const [allPatients, setAllPatients] = useState<HospitalPatientRecord[]>([]);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [error, setError] = useState("");
  const [selectedPatient, setSelectedPatient] = useState<HospitalPatientRecord | null>(null);
  const [loaded, setLoaded] = useState(false);

  const ensureLoaded = useCallback(async () => {
    if (loaded && allPatients.length > 0) return;
    setLoading(true);
    setError("");
    try {
      const res = await apiFetch("hospital", APIEndpoints.HOSPITAL_PATIENTS);
      const data = await res.json();
      if (!res.ok) {
        setError(data.error?.message || "Unable to load patients.");
        setAllPatients([]);
        return;
      }
      setAllPatients(data.data?.patients ?? data.patients ?? []);
      setLoaded(true);
    } catch {
      setError("Network error while loading patients.");
      setAllPatients([]);
    } finally {
      setLoading(false);
    }
  }, [allPatients.length, loaded]);

  useEffect(() => {
    if (!value) {
      setSelectedPatient(null);
      return;
    }
    const match = allPatients.find((p) => p.patient_id === value);
    if (match) setSelectedPatient(match);
  }, [value, allPatients]);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (event: MouseEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", onPointerDown);
    return () => document.removeEventListener("mousedown", onPointerDown);
  }, [open]);

  const filtered = useMemo(
    () => fuzzyFilterPatients(allPatients, query, 12),
    [allPatients, query],
  );

  const displaySelected =
    selectedPatient?.full_name ||
    selectedPatient?.email ||
    (value ? `Patient ${value.slice(0, 8)}…` : "");

  const openDropdown = async () => {
    setOpen(true);
    await ensureLoaded();
  };

  return (
    <div ref={rootRef} className={`relative space-y-2 ${className}`}>
      <label className="text-[length:var(--zh-body-size)] font-black uppercase tracking-wide text-[var(--zh-text-secondary)]">
        {label}
        {required ? " *" : ""}
      </label>

      {value && !open ? (
        <button
          type="button"
          className="flex w-full items-center gap-2 rounded-[var(--zh-radius-control)] border border-[var(--zh-success)] bg-[var(--zh-success-surface)]/70 px-3 py-2 text-left"
          onClick={() => {
            onChange("");
            setSelectedPatient(null);
            setQuery("");
            void openDropdown();
          }}
        >
          <div className="min-w-0 flex-1">
            <p className="truncate text-[length:var(--zh-body-size)] font-bold text-[var(--zh-text-primary)]">{displaySelected}</p>
            <p className="truncate text-[length:var(--zh-body-size)] font-semibold text-[var(--zh-text-secondary)]">
              {selectedPatient?.email || value}
            </p>
          </div>
          <span className="shrink-0 text-[length:var(--zh-body-size)] font-bold text-[var(--zh-text-secondary)]">Change</span>
        </button>
      ) : (
        <input
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setOpen(true);
          }}
          onFocus={() => {
            void openDropdown();
          }}
          placeholder="Search patient by name…"
          className="h-10 w-full rounded-[var(--zh-radius-control)] border border-[var(--zh-border-default)] bg-[var(--zh-surface-raised)] px-3 text-[length:var(--zh-body-size)] font-semibold text-[var(--zh-text-secondary)] outline-none focus:border-[var(--zh-info)] focus:ring-4 focus:ring-indigo-100"
          required={required && !value}
          autoComplete="off"
          role="combobox"
          aria-expanded={open}
          aria-controls="patient-picker-list"
        />
      )}

      {open ? (
        <div
          id="patient-picker-list"
          role="listbox"
          className="absolute z-30 mt-1 max-h-56 w-full overflow-auto rounded-[var(--zh-radius-control)] border border-[var(--zh-border-default)] bg-[var(--zh-surface-raised)] shadow-lg"
        >
          {loading ? (
            <p className="px-3 py-3 text-[length:var(--zh-body-size)] font-semibold text-[var(--zh-text-secondary)]">Loading…</p>
          ) : filtered.length === 0 ? (
            <p className="px-3 py-3 text-[length:var(--zh-body-size)] font-semibold text-[var(--zh-text-secondary)]">
              {error || (query.trim() ? "No patients matched." : "Start typing a name…")}
            </p>
          ) : (
            <ul>
              {filtered.map((patient) => (
                <li key={patient.patient_id}>
                  <button
                    type="button"
                    role="option"
                    aria-selected={value === (patient.patient_id || "")}
                    className="flex w-full flex-col items-start gap-0.5 px-3 py-2.5 text-left hover:bg-[var(--zh-surface-subtle)]"
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => {
                      setSelectedPatient(patient);
                      onChange(patient.patient_id || "", patient);
                      setOpen(false);
                      setQuery("");
                    }}
                  >
                    <span className="text-[length:var(--zh-body-size)] font-bold text-[var(--zh-text-primary)]">
                      {patient.full_name || "Unnamed patient"}
                    </span>
                    <span className="text-[length:var(--zh-body-size)] font-semibold text-[var(--zh-text-secondary)]">
                      {patient.email || "No email"}
                      {patient.phone_number ? ` · ${patient.phone_number}` : ""}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      ) : null}
    </div>
  );
}
