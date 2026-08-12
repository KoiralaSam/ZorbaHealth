"use client";

import { WEEKDAYS, WEEKDAY_FULL } from "@/components/AppointmentCalendar";

type Props = {
  activeWeekdays: number[];
  onToggle: (weekday: number) => void;
  startTime: string;
  endTime: string;
};

export function WeeklyScheduleBoard({
  activeWeekdays,
  onToggle,
  startTime,
  endTime,
}: Props) {
  const active = new Set(activeWeekdays);

  return (
    <div className="space-y-3">
      <div className="grid grid-cols-7 gap-2">
        {WEEKDAYS.map((short, weekday) => {
          const on = active.has(weekday);
          return (
            <button
              key={short}
              type="button"
              onClick={() => onToggle(weekday)}
              aria-pressed={on}
              title={`${WEEKDAY_FULL[weekday]} — tap to ${on ? "turn off" : "turn on"}`}
              className={[
                "flex min-h-[5.5rem] flex-col items-center justify-center gap-1 rounded-[var(--zh-radius-card)] border px-1 py-3 text-center transition",
                on
                  ? "border-[var(--zh-success)] bg-[var(--zh-success-surface)] text-emerald-900 shadow-sm"
                  : "border-[var(--zh-border-default)] bg-[var(--zh-surface-raised)] text-[var(--zh-text-secondary)] hover:border-[var(--zh-border-default)] hover:bg-[var(--zh-surface-subtle)]",
              ].join(" ")}
            >
              <span className="text-[length:var(--zh-body-size)] font-black uppercase tracking-wide">{short}</span>
              {on ? (
                <>
                  <span className="text-[11px] font-bold leading-tight">
                    {startTime}
                  </span>
                  <span className="text-[10px] font-semibold text-[var(--zh-success)]/80">to</span>
                  <span className="text-[11px] font-bold leading-tight">{endTime}</span>
                </>
              ) : (
                <span className="text-[11px] font-semibold">Off</span>
              )}
            </button>
          );
        })}
      </div>
      <p className="text-[length:var(--zh-body-size)] font-semibold text-[var(--zh-text-secondary)]">
        Green = available every week · Gray = off · Tap a day to flip it
      </p>
    </div>
  );
}
