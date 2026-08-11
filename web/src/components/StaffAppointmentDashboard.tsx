"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { apiFetch, clearApiCache } from "@/lib/auth-client";
import { APIEndpoints } from "@/contracts";
import {
  AppointmentCalendar,
  WEEKDAYS,
  formatSlotTime,
  toISODate,
} from "@/components/AppointmentCalendar";
import { PatientPicker } from "@/components/PatientPicker";
import { WeeklyScheduleBoard } from "@/components/WeeklyScheduleBoard";

type AvailabilityRule = {
  weekday: number;
  start_time_local: string;
  end_time_local: string;
  slot_duration_minutes: number;
  timezone: string;
};

type AvailabilityRuleApi = {
  weekday?: number;
  startTimeLocal?: string;
  start_time_local?: string;
  endTimeLocal?: string;
  end_time_local?: string;
  slotDurationMinutes?: number;
  slot_duration_minutes?: number;
  timezone?: string;
};

type Appointment = {
  id: string;
  patient_id: string;
  starts_at: string;
  status: string;
  title: string;
  type: string;
};

type Slot = { starts_at: string; ends_at: string; duration_minutes: number };

const TIME_OPTIONS = Array.from({ length: 24 * 2 }, (_, i) => {
  const h = Math.floor(i / 2);
  const m = i % 2 === 0 ? "00" : "30";
  return `${String(h).padStart(2, "0")}:${m}`;
});

export function StaffAppointmentDashboard() {
  const [tab, setTab] = useState<"appointments" | "availability" | "book">("appointments");
  const [appointments, setAppointments] = useState<Appointment[]>([]);
  const [activeWeekdays, setActiveWeekdays] = useState<number[]>([1, 2, 3, 4, 5]);
  const [startTime, setStartTime] = useState("09:00");
  const [endTime, setEndTime] = useState("17:00");
  const [slotMinutes, setSlotMinutes] = useState(30);
  const [timezone, setTimezone] = useState(
    typeof Intl !== "undefined" ? Intl.DateTimeFormat().resolvedOptions().timeZone : "UTC",
  );
  const [patientID, setPatientID] = useState("");
  const [slots, setSlots] = useState<Slot[]>([]);
  const [selectedDate, setSelectedDate] = useState("");
  const [selectedSlot, setSelectedSlot] = useState("");
  const [timeOffDate, setTimeOffDate] = useState(toISODate(new Date()));
  const [timeOffReason, setTimeOffReason] = useState("time off");
  const [availabilityMode, setAvailabilityMode] = useState<"weekly" | "timeoff">("weekly");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [loadingSlots, setLoadingSlots] = useState(false);
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
    const res = await apiFetch("hospital", APIEndpoints.HOSPITAL_APPOINTMENTS);
    const data = await res.json();
    setAppointments(data.data?.appointments ?? data.appointments ?? []);
  }, []);

  const loadAvailability = useCallback(async () => {
    const res = await apiFetch("hospital", APIEndpoints.HOSPITAL_AVAILABILITY);
    const data = await res.json();
    const loaded = (data.data?.rules ?? data.rules ?? []) as AvailabilityRuleApi[];
    if (!Array.isArray(loaded) || loaded.length === 0) return;
    const mapped: AvailabilityRule[] = loaded.map((r) => ({
      weekday: r.weekday ?? 1,
      start_time_local: r.startTimeLocal || r.start_time_local || "09:00",
      end_time_local: r.endTimeLocal || r.end_time_local || "17:00",
      slot_duration_minutes: r.slotDurationMinutes || r.slot_duration_minutes || 30,
      timezone: r.timezone || "UTC",
    }));
    setActiveWeekdays([...new Set(mapped.map((r) => r.weekday))].sort());
    setStartTime(mapped[0].start_time_local);
    setEndTime(mapped[0].end_time_local);
    setSlotMinutes(mapped[0].slot_duration_minutes);
    setTimezone(mapped[0].timezone || timezone);
  }, [timezone]);

  const loadSlots = useCallback(async () => {
    setLoadingSlots(true);
    try {
      const from = new Date().toISOString();
      const to = new Date(Date.now() + 28 * 86400000).toISOString();
      const qs = new URLSearchParams({ from, to });
      const res = await apiFetch("hospital", `${APIEndpoints.HOSPITAL_APPOINTMENT_SLOTS}?${qs}`);
      const data = await res.json();
      const list: Slot[] = data.data?.slots ?? data.slots ?? [];
      setSlots(list);
      if (list.length > 0) {
        const firstDate = list[0].starts_at.slice(0, 10);
        setSelectedDate(firstDate);
        setSelectedSlot(list[0].starts_at);
      }
    } finally {
      setLoadingSlots(false);
    }
  }, []);

  useEffect(() => {
    void loadAppointments();
    void loadAvailability();
  }, [loadAppointments, loadAvailability]);

  useEffect(() => {
    if (tab === "book") {
      void loadSlots();
    }
  }, [tab, loadSlots]);

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

  const toggleWeekday = (day: number) => {
    setActiveWeekdays((prev) =>
      prev.includes(day) ? prev.filter((d) => d !== day) : [...prev, day].sort(),
    );
  };

  const applyWeekdayPreset = () => {
    setActiveWeekdays([1, 2, 3, 4, 5]);
    setStartTime("09:00");
    setEndTime("17:00");
    setSlotMinutes(30);
  };

  const saveAvailability = async () => {
    setBusy(true);
    setError("");
    setMessage("");
    try {
      const rules: AvailabilityRule[] = activeWeekdays.map((weekday) => ({
        weekday,
        start_time_local: startTime,
        end_time_local: endTime,
        slot_duration_minutes: slotMinutes,
        timezone,
      }));
      const res = await apiFetch("hospital", APIEndpoints.HOSPITAL_AVAILABILITY, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ rules }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error?.message || "Failed to save availability");
        return;
      }
      setMessage("Weekly availability saved.");
    } finally {
      setBusy(false);
    }
  };

  const addException = async () => {
    setBusy(true);
    setError("");
    setMessage("");
    try {
      const start = new Date(`${timeOffDate}T00:00:00`);
      const end = new Date(`${timeOffDate}T23:59:59`);
      const res = await apiFetch("hospital", APIEndpoints.HOSPITAL_AVAILABILITY_EXCEPTIONS, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          starts_at: start.toISOString(),
          ends_at: end.toISOString(),
          reason: timeOffReason,
          is_available: false,
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error?.message || "Failed to add time off");
        return;
      }
      setMessage(`Time off added for ${timeOffDate}.`);
      if (tab === "book") void loadSlots();
    } finally {
      setBusy(false);
    }
  };

  const bookForPatient = async () => {
    setBusy(true);
    setError("");
    setMessage("");
    try {
      const res = await apiFetch("hospital", APIEndpoints.HOSPITAL_APPOINTMENTS, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          patient_id: patientID.trim(),
          starts_at: selectedSlot,
          duration_minutes: slotMinutes,
          timezone,
          type: "video",
          title: "Zorba Health appointment",
          send_email: notifyEmail,
          send_sms: notifySMS,
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error?.message || "Failed to book");
        return;
      }
      clearApiCache("hospital");
      mergeAppointment(data.data?.appointment ?? data.appointment);
      const channels = [
        notifyEmail ? "email" : null,
        notifySMS ? "SMS" : null,
      ].filter(Boolean);
      setMessage(
        channels.length
          ? `Appointment booked. Confirmation will be sent by ${channels.join(" and ")}.`
          : "Appointment booked for patient.",
      );
      setTab("appointments");
      await loadAppointments();
      await loadSlots();
    } finally {
      setBusy(false);
    }
  };

  const cancel = async (id: string) => {
    setBusy(true);
    setError("");
    try {
      const res = await apiFetch("hospital", `${APIEndpoints.HOSPITAL_APPOINTMENTS}/${encodeURIComponent(id)}`, {
        method: "DELETE",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ reason: "cancelled_by_staff" }),
      });
      const data = await res.json();
      if (!res.ok) {
        setError(data.error?.message || "Unable to cancel appointment.");
        return;
      }
      clearApiCache("hospital");
      const updated: Appointment | undefined = data.data?.appointment ?? data.appointment;
      if (updated) {
        mergeAppointment(updated);
      } else {
        setAppointments((prev) =>
          prev.map((item) => (item.id === id ? { ...item, status: "cancelled" } : item)),
        );
      }
      setMessage("Appointment cancelled.");
      if (tab === "book") void loadSlots();
    } finally {
      setBusy(false);
    }
  };

  return (
    <section className="clinical-card space-y-5 p-6">
      <div className="flex flex-wrap gap-2">
        {(["appointments", "availability", "book"] as const).map((t) => (
          <button
            key={t}
            type="button"
            className={`rounded-full px-4 py-1.5 text-sm font-bold ${tab === t ? "bg-slate-900 text-white" : "bg-slate-100"}`}
            onClick={() => setTab(t)}
          >
            {t === "appointments" ? "Appointments" : t === "availability" ? "Availability" : "Book for patient"}
          </button>
        ))}
      </div>

      {error ? <p className="text-sm text-rose-600">{error}</p> : null}
      {message ? <p className="text-sm text-emerald-700">{message}</p> : null}

      {tab === "appointments" ? (
        <ul className="space-y-2">
          {appointments.map((a) => (
            <li
              key={a.id}
              className={`flex items-center justify-between rounded-xl border px-3 py-2 text-sm ${
                a.status === "cancelled" ? "border-slate-200 bg-slate-50 text-slate-500" : ""
              }`}
            >
              <span>
                {a.title} · patient {(a.patient_id || "").slice(0, 8) || "—"}… ·{" "}
                {new Date(a.starts_at).toLocaleString()} · {a.status}
              </span>
              {a.status === "booked" ? (
                <button type="button" className="text-rose-600" onClick={() => void cancel(a.id)}>
                  Cancel
                </button>
              ) : null}
            </li>
          ))}
          {appointments.length === 0 ? <li className="text-slate-500">No appointments.</li> : null}
        </ul>
      ) : null}

      {tab === "availability" ? (
        <div className="space-y-5">
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              className={`rounded-full px-4 py-1.5 text-sm font-bold ${
                availabilityMode === "weekly" ? "bg-slate-900 text-white" : "bg-slate-100"
              }`}
              onClick={() => setAvailabilityMode("weekly")}
            >
              1. Weekly hours
            </button>
            <button
              type="button"
              className={`rounded-full px-4 py-1.5 text-sm font-bold ${
                availabilityMode === "timeoff" ? "bg-slate-900 text-white" : "bg-slate-100"
              }`}
              onClick={() => setAvailabilityMode("timeoff")}
            >
              2. Time off
            </button>
          </div>

          {availabilityMode === "weekly" ? (
            <div className="space-y-5">
              <div className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm text-slate-700">
                <p className="font-bold text-slate-900">How to set availability</p>
                <ol className="mt-2 list-decimal space-y-1 pl-5 font-semibold">
                  <li>Tap each day of the week you work (green = on, gray = off).</li>
                  <li>Choose the start and end time for those days.</li>
                  <li>Press Save weekly hours.</li>
                </ol>
              </div>

              <div className="flex flex-wrap gap-2">
                <button
                  type="button"
                  className="rounded-full bg-emerald-50 px-3 py-1.5 text-xs font-bold text-emerald-800 ring-1 ring-emerald-200"
                  onClick={applyWeekdayPreset}
                >
                  Use Mon–Fri 9:00–17:00
                </button>
                <button
                  type="button"
                  className="rounded-full bg-slate-100 px-3 py-1.5 text-xs font-bold text-slate-600"
                  onClick={() => setActiveWeekdays([])}
                >
                  Clear days
                </button>
              </div>

              <WeeklyScheduleBoard
                activeWeekdays={activeWeekdays}
                onToggle={toggleWeekday}
                startTime={startTime}
                endTime={endTime}
              />

              <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
                <label className="text-sm font-semibold">
                  Start time
                  <select
                    className="mt-1 w-full rounded-xl border px-3 py-2"
                    value={startTime}
                    onChange={(e) => setStartTime(e.target.value)}
                  >
                    {TIME_OPTIONS.map((t) => (
                      <option key={t} value={t}>
                        {t}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="text-sm font-semibold">
                  End time
                  <select
                    className="mt-1 w-full rounded-xl border px-3 py-2"
                    value={endTime}
                    onChange={(e) => setEndTime(e.target.value)}
                  >
                    {TIME_OPTIONS.map((t) => (
                      <option key={t} value={t}>
                        {t}
                      </option>
                    ))}
                  </select>
                </label>
                <label className="text-sm font-semibold">
                  Appointment length
                  <select
                    className="mt-1 w-full rounded-xl border px-3 py-2"
                    value={slotMinutes}
                    onChange={(e) => setSlotMinutes(Number(e.target.value))}
                  >
                    {[15, 20, 30, 45, 60].map((m) => (
                      <option key={m} value={m}>
                        {m} min
                      </option>
                    ))}
                  </select>
                </label>
                <label className="text-sm font-semibold">
                  Timezone
                  <input
                    className="mt-1 w-full rounded-xl border px-3 py-2"
                    value={timezone}
                    onChange={(e) => setTimezone(e.target.value)}
                  />
                </label>
              </div>

              <div className="flex flex-wrap items-center gap-3">
                <button
                  type="button"
                  className="rounded-full bg-slate-900 px-5 py-2.5 text-sm font-bold text-white disabled:opacity-50"
                  disabled={busy || activeWeekdays.length === 0 || startTime >= endTime}
                  onClick={() => void saveAvailability()}
                >
                  {busy ? "Saving…" : "Save weekly hours"}
                </button>
                <p className="text-sm font-semibold text-slate-600">
                  {activeWeekdays.length === 0
                    ? "Select at least one working day."
                    : startTime >= endTime
                      ? "End time must be after start time."
                      : `Ready: ${activeWeekdays.map((d) => WEEKDAYS[d]).join(", ")} · ${startTime}–${endTime}`}
                </p>
              </div>
            </div>
          ) : (
            <div className="grid gap-6 lg:grid-cols-2">
              <div className="space-y-3">
                <div className="rounded-2xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-950">
                  <p className="font-bold">Block a specific day off</p>
                  <p className="mt-1 font-semibold">
                    Pick a date on the calendar, then save. That day will not offer booking slots.
                  </p>
                </div>
                <AppointmentCalendar
                  label="Select the day you are unavailable"
                  value={timeOffDate}
                  onChange={setTimeOffDate}
                  mode="any"
                />
              </div>
              <div className="space-y-3">
                <p className="text-sm font-bold text-slate-900">
                  Time off on {timeOffDate || "—"}
                </p>
                <input
                  className="w-full rounded-xl border px-3 py-2 text-sm"
                  value={timeOffReason}
                  onChange={(e) => setTimeOffReason(e.target.value)}
                  placeholder="Reason (optional)"
                />
                <button
                  type="button"
                  className="rounded-full bg-amber-600 px-5 py-2.5 text-sm font-bold text-white disabled:opacity-50"
                  disabled={busy || !timeOffDate}
                  onClick={() => void addException()}
                >
                  {busy ? "Saving…" : "Save time off"}
                </button>
              </div>
            </div>
          )}
        </div>
      ) : null}

      {tab === "book" ? (
        <div className="grid gap-6 lg:grid-cols-2">
          <div className="space-y-3">
            <PatientPicker
              value={patientID}
              onChange={(id) => setPatientID(id)}
              label="Patient"
              required
            />
            <AppointmentCalendar
              label="Pick an available date"
              value={selectedDate}
              onChange={setSelectedDate}
              availableDates={availableDates}
              mode="available"
            />
            {loadingSlots ? <p className="text-sm text-slate-500">Loading openings…</p> : null}
          </div>
          <div className="space-y-3">
            <p className="text-sm font-bold">Times on {selectedDate || "—"}</p>
            <div className="flex flex-wrap gap-2">
              {slotsForDate.map((s) => {
                const selected = selectedSlot === s.starts_at;
                return (
                  <button
                    key={s.starts_at}
                    type="button"
                    onClick={() => setSelectedSlot(s.starts_at)}
                    className={`rounded-full px-4 py-2 text-sm font-bold ${
                      selected ? "bg-slate-900 text-white" : "bg-emerald-50 text-emerald-900 ring-1 ring-emerald-200"
                    }`}
                  >
                    {formatSlotTime(s.starts_at)}
                  </button>
                );
              })}
              {!loadingSlots && selectedDate && slotsForDate.length === 0 ? (
                <p className="text-sm text-slate-500">No openings on this date.</p>
              ) : null}
            </div>
            <div className="space-y-2 text-sm">
              <p className="font-bold text-slate-800">Send confirmation</p>
              <label className="flex items-center gap-2 font-semibold text-slate-700">
                <input
                  type="checkbox"
                  checked={notifyEmail}
                  onChange={(e) => setNotifyEmail(e.target.checked)}
                />
                Email
              </label>
              <label className="flex items-center gap-2 font-semibold text-slate-700">
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
              className="rounded-full bg-slate-900 px-5 py-2 text-sm font-bold text-white disabled:opacity-50"
              disabled={busy || !patientID || !selectedSlot}
              onClick={() => void bookForPatient()}
            >
              Book selected time
            </button>
          </div>
        </div>
      ) : null}
    </section>
  );
}
