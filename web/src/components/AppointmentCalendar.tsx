"use client";

import { useMemo, useState } from "react";

const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
const WEEKDAY_FULL = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

function toISODate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function startOfMonth(d: Date): Date {
  return new Date(d.getFullYear(), d.getMonth(), 1);
}

function addMonths(d: Date, n: number): Date {
  return new Date(d.getFullYear(), d.getMonth() + n, 1);
}

function parseISODate(iso: string): Date {
  const [y, m, day] = iso.split("-").map(Number);
  return new Date(y, m - 1, day);
}

export type CalendarMode = "available" | "any" | "weekday";

type Props = {
  value: string;
  onChange: (isoDate: string) => void;
  availableDates?: string[];
  /** Weekdays (0=Sun..6=Sat) that are part of the weekly schedule. Used in mode="weekday". */
  activeWeekdays?: number[];
  /** Called when a calendar day is tapped in weekday mode — toggles that weekday. */
  onToggleWeekday?: (weekday: number, isoDate: string) => void;
  mode?: CalendarMode;
  className?: string;
  label?: string;
  disabled?: boolean;
  disabledText?: string;
};

export function AppointmentCalendar({
  value,
  onChange,
  availableDates = [],
  activeWeekdays = [],
  onToggleWeekday,
  mode = "any",
  className = "",
  label,
  disabled = false,
  disabledText = "Select a hospital and staff member to see open dates.",
}: Props) {
  const availableSet = useMemo(() => new Set(availableDates), [availableDates]);
  const activeWeekdaySet = useMemo(() => new Set(activeWeekdays), [activeWeekdays]);
  const initial = value ? parseISODate(value) : new Date();
  const [cursor, setCursor] = useState(startOfMonth(initial));

  const cells = useMemo(() => {
    const first = startOfMonth(cursor);
    const startPad = first.getDay();
    const daysInMonth = new Date(cursor.getFullYear(), cursor.getMonth() + 1, 0).getDate();
    const out: Array<{ iso: string; day: number; weekday: number } | null> = [];
    for (let i = 0; i < startPad; i++) out.push(null);
    for (let day = 1; day <= daysInMonth; day++) {
      const d = new Date(cursor.getFullYear(), cursor.getMonth(), day);
      out.push({ iso: toISODate(d), day, weekday: d.getDay() });
    }
    while (out.length % 7 !== 0) out.push(null);
    return out;
  }, [cursor]);

  const monthLabel = cursor.toLocaleString(undefined, { month: "long", year: "numeric" });
  const todayISO = toISODate(new Date());

  return (
    <div
      className={`rounded-2xl border border-slate-200 bg-white p-4 shadow-sm ${
        disabled ? "opacity-50" : ""
      } ${className}`}
    >
      {label ? <p className="mb-3 text-sm font-bold text-slate-800">{label}</p> : null}
      <div className="mb-3 flex items-center justify-between">
        <button
          type="button"
          className="rounded-full px-3 py-1 text-sm font-bold text-slate-700 hover:bg-slate-100 disabled:cursor-not-allowed"
          onClick={() => setCursor((c) => addMonths(c, -1))}
          aria-label="Previous month"
          disabled={disabled}
        >
          ‹
        </button>
        <p className="text-sm font-black text-slate-900">{monthLabel}</p>
        <button
          type="button"
          className="rounded-full px-3 py-1 text-sm font-bold text-slate-700 hover:bg-slate-100 disabled:cursor-not-allowed"
          onClick={() => setCursor((c) => addMonths(c, 1))}
          aria-label="Next month"
          disabled={disabled}
        >
          ›
        </button>
      </div>
      <div className="mb-1 grid grid-cols-7 gap-1 text-center text-[11px] font-bold uppercase tracking-wide text-slate-400">
        {WEEKDAYS.map((d, i) => (
          <div
            key={d}
            className={
              mode === "weekday" && activeWeekdaySet.has(i) ? "text-emerald-700" : undefined
            }
          >
            {d}
          </div>
        ))}
      </div>
      <div className="grid grid-cols-7 gap-1">
        {cells.map((cell, idx) => {
          if (!cell) {
            return <div key={`pad-${idx}`} className="h-10" />;
          }
          const isAvailable = !disabled && availableSet.has(cell.iso);
          const isWorkingWeekday = mode === "weekday" && activeWeekdaySet.has(cell.weekday);
          const isPast = cell.iso < todayISO;
          const selectable = disabled
            ? false
            : mode === "weekday"
              ? true
              : mode === "any"
                ? !isPast
                : isAvailable && !isPast;
          const selected = !disabled && value === cell.iso;
          return (
            <button
              key={cell.iso}
              type="button"
              disabled={!selectable}
              onClick={() => {
                if (mode === "weekday") {
                  onToggleWeekday?.(cell.weekday, cell.iso);
                  onChange(cell.iso);
                  return;
                }
                onChange(cell.iso);
              }}
              className={[
                "h-10 rounded-xl text-sm font-bold transition",
                selected
                  ? "bg-slate-900 text-white shadow"
                  : mode === "weekday" && isWorkingWeekday
                    ? "bg-emerald-50 text-emerald-800 ring-1 ring-emerald-200 hover:bg-emerald-100"
                    : selectable
                      ? isAvailable
                        ? "bg-emerald-50 text-emerald-800 ring-1 ring-emerald-200 hover:bg-emerald-100"
                        : "bg-slate-50 text-slate-800 hover:bg-slate-100"
                      : "cursor-not-allowed text-slate-300",
              ].join(" ")}
              title={
                disabled
                  ? disabledText
                  : mode === "weekday"
                    ? isWorkingWeekday
                      ? `${WEEKDAY_FULL[cell.weekday]} — working day (tap to remove)`
                      : `${WEEKDAY_FULL[cell.weekday]} — off (tap to add)`
                    : isAvailable
                      ? "Available"
                      : mode === "available"
                        ? "No openings"
                        : cell.iso
              }
            >
              {cell.day}
            </button>
          );
        })}
      </div>
      {disabled ? (
        <p className="mt-3 text-xs text-slate-500">{disabledText}</p>
      ) : mode === "weekday" ? (
        <p className="mt-3 text-xs text-slate-500">
          Tap any date to turn that weekday on or off for every week. Green days are working days.
        </p>
      ) : mode === "available" ? (
        <p className="mt-3 text-xs text-slate-500">
          Green dates have open slots. Tap one to auto-select the earliest time.
        </p>
      ) : null}
    </div>
  );
}

export function formatSlotTime(iso: string): string {
  try {
    return new Date(iso).toLocaleTimeString(undefined, {
      hour: "numeric",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
}

export { WEEKDAYS, WEEKDAY_FULL, toISODate };
