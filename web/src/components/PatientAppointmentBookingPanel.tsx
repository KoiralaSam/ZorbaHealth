"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { apiFetch, clearApiCache } from "@/lib/auth-client";
import { APIEndpoints } from "@/contracts";
import { AppointmentCalendar, formatSlotTime } from "@/components/AppointmentCalendar";

type AppointmentSlot = {
  starts_at: string;
  ends_at: string;
  duration_minutes: number;
  timezone: string;
  staff_id: string;
  hospital_id: string;
};

type Appointment = {
  id: string;
  patient_id: string;
  staff_id: string;
  hospital_id: string;
  starts_at: string;
  duration_minutes: number;
  timezone: string;
  type: string;
  status: string;
  title: string;
  join_url?: string;
};

type StaffOption = { staff_id: string; name: string; role: string };
type HospitalOption = { hospital_id: string; hospital_name: string };

type Props = {
  hospitals: HospitalOption[];
  loadStaff: (hospitalID: string) => Promise<StaffOption[]>;
};

export function PatientAppointmentBookingPanel({ hospitals, loadStaff }: Props) {
  const [appointments, setAppointments] = useState<Appointment[]>([]);
  const [staff, setStaff] = useState<StaffOption[]>([]);
  const [slots, setSlots] = useState<AppointmentSlot[]>([]);
  const [hospitalID, setHospitalID] = useState("");
  const [staffID, setStaffID] = useState("");
  const [selectedDate, setSelectedDate] = useState("");
  const [selectedSlot, setSelectedSlot] = useState("");
  const [timezone, setTimezone] = useState(
    typeof Intl !== "undefined" ? Intl.DateTimeFormat().resolvedOptions().timeZone : "UTC",
  );
  const [busy, setBusy] = useState(false);
  const [loadingSlots, setLoadingSlots] = useState(false);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");
  const [notifyEmail, setNotifyEmail] = useState(true);
  const [notifySMS, setNotifySMS] = useState(false);

  const mergeAppointment = useCallback((appointment?: Appointment) => {
    if (!appointment?.id) return;
    setAppointments((current) => {
      const exists = current.some((item) => item.id === appointment.id);
      if (!exists) return [appointment, ...current];
      return current.map((item) => (item.id === appointment.id ? appointment : item));
    });
  }, []);

  const loadAppointments = useCallback(async () => {
    const res = await apiFetch("patient", APIEndpoints.PATIENT_APPOINTMENTS);
    const data = await res.json();
    setAppointments(data.data?.appointments ?? data.appointments ?? []);
  }, []);

  useEffect(() => {
    void loadAppointments();
  }, [loadAppointments]);

  useEffect(() => {
    if (!hospitalID) {
      setStaff([]);
      return;
    }
    void loadStaff(hospitalID).then(setStaff);
  }, [hospitalID, loadStaff]);

  useEffect(() => {
    if (!hospitalID || !staffID) {
      setSlots([]);
      setSelectedDate("");
      setSelectedSlot("");
      return;
    }
    setLoadingSlots(true);
    const from = new Date().toISOString();
    const to = new Date(Date.now() + 28 * 86400000).toISOString();
    const qs = new URLSearchParams({
      staff_id: staffID,
      hospital_id: hospitalID,
      from,
      to,
    });
    void apiFetch("patient", `${APIEndpoints.PATIENT_APPOINTMENT_SLOTS}?${qs}`)
      .then(async (res) => {
        const data = await res.json();
        const list: AppointmentSlot[] = data.data?.slots ?? data.slots ?? [];
        setSlots(list);
        if (list.length > 0) {
          const firstDate = list[0].starts_at.slice(0, 10);
          setSelectedDate(firstDate);
          setSelectedSlot(list[0].starts_at);
          if (list[0].timezone) setTimezone(list[0].timezone);
        } else {
          setSelectedDate("");
          setSelectedSlot("");
        }
      })
      .finally(() => setLoadingSlots(false));
  }, [hospitalID, staffID]);

  const availableDates = useMemo(
    () => [...new Set(slots.map((s) => s.starts_at.slice(0, 10)))].sort(),
    [slots],
  );

  const slotsForDate = useMemo(
    () => (selectedDate ? slots.filter((s) => s.starts_at.slice(0, 10) === selectedDate) : []),
    [slots, selectedDate],
  );

  useEffect(() => {
    if (slotsForDate.length === 0) {
      setSelectedSlot("");
      return;
    }
    if (!slotsForDate.some((s) => s.starts_at === selectedSlot)) {
      setSelectedSlot(slotsForDate[0].starts_at);
    }
  }, [slotsForDate, selectedSlot]);

  const book = async () => {
    setBusy(true);
    setError("");
    setMessage("");
    try {
      const res = await apiFetch("patient", APIEndpoints.PATIENT_APPOINTMENTS, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          staff_id: staffID,
          hospital_id: hospitalID,
          starts_at: selectedSlot,
          duration_minutes: 30,
          timezone,
          type: "video",
          title: "Zorba Health appointment",
          send_email: notifyEmail,
          send_sms: notifySMS,
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error?.message || "Unable to book appointment.");
        return;
      }
      clearApiCache("patient");
      mergeAppointment(data.data?.appointment ?? data.appointment);
      const channels = [
        notifyEmail ? "email" : null,
        notifySMS ? "SMS" : null,
      ].filter(Boolean);
      setMessage(
        channels.length
          ? `Appointment booked. Confirmation will be sent by ${channels.join(" and ")}.`
          : "Appointment booked.",
      );
      setSelectedSlot("");
      // Refresh openings so the booked slot disappears from the calendar.
      if (hospitalID && staffID) {
        const from = new Date().toISOString();
        const to = new Date(Date.now() + 28 * 86400000).toISOString();
        const qs = new URLSearchParams({
          staff_id: staffID,
          hospital_id: hospitalID,
          from,
          to,
        });
        const slotsRes = await apiFetch("patient", `${APIEndpoints.PATIENT_APPOINTMENT_SLOTS}?${qs}`);
        const slotsData = await slotsRes.json();
        setSlots(slotsData.data?.slots ?? slotsData.slots ?? []);
      }
      await loadAppointments();
    } finally {
      setBusy(false);
    }
  };

  const cancel = async (id: string) => {
    setBusy(true);
    setError("");
    try {
      const res = await apiFetch("patient", `${APIEndpoints.PATIENT_APPOINTMENTS}/${encodeURIComponent(id)}`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reason: "cancelled_by_patient" }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error?.message || "Unable to cancel appointment.");
        return;
      }
      clearApiCache("patient");
      const updated: Appointment | undefined = data.data?.appointment ?? data.appointment;
      if (updated) {
        mergeAppointment(updated);
      } else {
        setAppointments((prev) =>
          prev.map((item) => (item.id === id ? { ...item, status: "cancelled" } : item)),
        );
      }
      setMessage("Appointment cancelled.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="clinical-card space-y-6 p-6">
      <div>
<<<<<<< HEAD
        <h2 className="text-[length:var(--zh-heading-size)] font-black">Book appointment</h2>
        <p className="mt-1 text-[length:var(--zh-body-size)] text-[var(--zh-text-secondary)]">
=======
        <h2 className="text-lg font-black">Book appointment</h2>
        <p className="mt-1 text-sm text-slate-600">
>>>>>>> af5074b (Sync active ZorbaHealth changes)
          Choose your care team, then tap a green calendar date. The earliest open time is selected automatically.
        </p>
      </div>

      <div className="grid gap-3 md:grid-cols-2">
<<<<<<< HEAD
        <label className="text-[length:var(--zh-body-size)] font-semibold">
          Hospital
          <select
            className="mt-1 w-full rounded-[var(--zh-radius-control)] border px-3 py-2"
=======
        <label className="text-sm font-semibold">
          Hospital
          <select
            className="mt-1 w-full rounded-xl border px-3 py-2"
>>>>>>> af5074b (Sync active ZorbaHealth changes)
            value={hospitalID}
            onChange={(e) => {
              setHospitalID(e.target.value);
              setStaffID("");
            }}
          >
            <option value="">Select hospital</option>
            {hospitals.map((h) => (
              <option key={h.hospital_id} value={h.hospital_id}>
                {h.hospital_name}
              </option>
            ))}
          </select>
        </label>
<<<<<<< HEAD
        <label className="text-[length:var(--zh-body-size)] font-semibold">
          Doctor / staff
          <select
            className="mt-1 w-full rounded-[var(--zh-radius-control)] border px-3 py-2"
=======
        <label className="text-sm font-semibold">
          Doctor / staff
          <select
            className="mt-1 w-full rounded-xl border px-3 py-2"
>>>>>>> af5074b (Sync active ZorbaHealth changes)
            value={staffID}
            onChange={(e) => setStaffID(e.target.value)}
            disabled={!hospitalID}
          >
            <option value="">Select staff</option>
            {staff.map((s) => (
              <option key={s.staff_id} value={s.staff_id}>
                {s.name} ({s.role})
              </option>
            ))}
          </select>
        </label>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <AppointmentCalendar
          label="Available dates"
          value={selectedDate}
          onChange={setSelectedDate}
          availableDates={availableDates}
          mode="available"
          disabled={!hospitalID || !staffID}
          disabledText="Select a hospital and doctor to see open dates."
        />
        <div className={`space-y-3 ${!hospitalID || !staffID ? "opacity-50" : ""}`}>
<<<<<<< HEAD
          <p className="text-[length:var(--zh-body-size)] font-bold">
=======
          <p className="text-sm font-bold">
>>>>>>> af5074b (Sync active ZorbaHealth changes)
            {!hospitalID || !staffID ? "Times" : `Times on ${selectedDate || "—"}`}
            {hospitalID && staffID && loadingSlots ? " · loading…" : ""}
          </p>
          <div className="flex flex-wrap gap-2">
            {slotsForDate.map((s) => {
              const selected = selectedSlot === s.starts_at;
              return (
                <button
                  key={s.starts_at}
                  type="button"
                  onClick={() => setSelectedSlot(s.starts_at)}
                  disabled={!hospitalID || !staffID}
<<<<<<< HEAD
                  className={`rounded-[var(--zh-radius-pill)] px-4 py-2 text-[length:var(--zh-body-size)] font-bold ${
                    selected
                      ? "bg-[var(--zh-surface-raised)] text-white"
                      : "bg-[var(--zh-success-surface)] text-emerald-900 ring-1 ring-emerald-200"
=======
                  className={`rounded-full px-4 py-2 text-sm font-bold ${
                    selected
                      ? "bg-slate-900 text-white"
                      : "bg-emerald-50 text-emerald-900 ring-1 ring-emerald-200"
>>>>>>> af5074b (Sync active ZorbaHealth changes)
                  }`}
                >
                  {formatSlotTime(s.starts_at)}
                </button>
              );
            })}
            {hospitalID && staffID && !loadingSlots && selectedDate && slotsForDate.length === 0 ? (
<<<<<<< HEAD
              <p className="text-[length:var(--zh-body-size)] text-[var(--zh-text-secondary)]">No openings on this date.</p>
            ) : null}
            {hospitalID && staffID && !loadingSlots && availableDates.length === 0 ? (
              <p className="text-[length:var(--zh-body-size)] text-[var(--zh-text-secondary)]">
=======
              <p className="text-sm text-slate-500">No openings on this date.</p>
            ) : null}
            {hospitalID && staffID && !loadingSlots && availableDates.length === 0 ? (
              <p className="text-sm text-slate-500">
>>>>>>> af5074b (Sync active ZorbaHealth changes)
                No openings in the next 4 weeks. Ask your care team to set availability.
              </p>
            ) : null}
            {!hospitalID || !staffID ? (
<<<<<<< HEAD
              <p className="text-[length:var(--zh-body-size)] text-[var(--zh-text-secondary)]">
=======
              <p className="text-sm text-slate-500">
>>>>>>> af5074b (Sync active ZorbaHealth changes)
                Select a hospital and doctor to load available times.
              </p>
            ) : null}
          </div>
<<<<<<< HEAD
            <div className="space-y-2 text-[length:var(--zh-body-size)]">
              <p className="font-bold text-[var(--zh-text-primary)]">Send confirmation</p>
              <label className="flex items-center gap-2 font-semibold text-[var(--zh-text-secondary)]">
=======
            <div className="space-y-2 text-sm">
              <p className="font-bold text-slate-800">Send confirmation</p>
              <label className="flex items-center gap-2 font-semibold text-slate-700">
>>>>>>> af5074b (Sync active ZorbaHealth changes)
                <input
                  type="checkbox"
                  checked={notifyEmail}
                  onChange={(e) => setNotifyEmail(e.target.checked)}
                />
                Email
              </label>
<<<<<<< HEAD
              <label className="flex items-center gap-2 font-semibold text-[var(--zh-text-secondary)]">
=======
              <label className="flex items-center gap-2 font-semibold text-slate-700">
>>>>>>> af5074b (Sync active ZorbaHealth changes)
                <input
                  type="checkbox"
                  checked={notifySMS}
                  onChange={(e) => setNotifySMS(e.target.checked)}
                />
                SMS
              </label>
            </div>
          <button
            type="button"
<<<<<<< HEAD
            className="rounded-[var(--zh-radius-pill)] bg-[var(--zh-surface-raised)] px-5 py-2 text-[length:var(--zh-body-size)] font-bold text-white disabled:opacity-50"
=======
            className="rounded-full bg-slate-900 px-5 py-2 text-sm font-bold text-white disabled:opacity-50"
>>>>>>> af5074b (Sync active ZorbaHealth changes)
            disabled={busy || !selectedSlot || !hospitalID || !staffID}
            onClick={() => void book()}
          >
            {busy ? "Booking…" : "Confirm appointment"}
          </button>
        </div>
      </div>

<<<<<<< HEAD
      {error ? <p className="text-[length:var(--zh-body-size)] text-rose-600">{error}</p> : null}
      {message ? <p className="text-[length:var(--zh-body-size)] text-[var(--zh-success)]">{message}</p> : null}
=======
      {error ? <p className="text-sm text-rose-600">{error}</p> : null}
      {message ? <p className="text-sm text-emerald-700">{message}</p> : null}
>>>>>>> af5074b (Sync active ZorbaHealth changes)

      <div className="border-t pt-4">
        <h3 className="font-bold">Upcoming appointments</h3>
        <ul className="mt-3 space-y-2">
          {appointments.map((a) => (
            <li
              key={a.id}
<<<<<<< HEAD
              className={`flex items-center justify-between rounded-[var(--zh-radius-control)] border px-3 py-2 text-[length:var(--zh-body-size)] ${
                a.status === "cancelled" ? "border-[var(--zh-border-default)] bg-[var(--zh-surface-subtle)] text-[var(--zh-text-secondary)]" : ""
=======
              className={`flex items-center justify-between rounded-xl border px-3 py-2 text-sm ${
                a.status === "cancelled" ? "border-slate-200 bg-slate-50 text-slate-500" : ""
>>>>>>> af5074b (Sync active ZorbaHealth changes)
              }`}
            >
              <span>
                {a.title} · {new Date(a.starts_at).toLocaleString()} · {a.status}
              </span>
              {a.status === "booked" ? (
                <button type="button" className="text-rose-600" onClick={() => void cancel(a.id)}>
                  Cancel
                </button>
              ) : null}
            </li>
          ))}
<<<<<<< HEAD
          {appointments.length === 0 ? <li className="text-[var(--zh-text-secondary)]">No appointments yet.</li> : null}
=======
          {appointments.length === 0 ? <li className="text-slate-500">No appointments yet.</li> : null}
>>>>>>> af5074b (Sync active ZorbaHealth changes)
        </ul>
      </div>
    </section>
  );
}
