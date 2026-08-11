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
                "flex min-h-[5.5rem] flex-col items-center justify-center gap-1 rounded-2xl border px-1 py-3 text-center transition",
                on
                  ? "border-emerald-300 bg-emerald-50 text-emerald-900 shadow-sm"
                  : "border-slate-200 bg-white text-slate-400 hover:border-slate-300 hover:bg-slate-50",
              ].join(" ")}
            >
              <span className="text-xs font-black uppercase tracking-wide">{short}</span>
              {on ? (
                <>
                  <span className="text-[11px] font-bold leading-tight">
                    {startTime}
                  </span>
                  <span className="text-[10px] font-semibold text-emerald-700/80">to</span>
                  <span className="text-[11px] font-bold leading-tight">{endTime}</span>
                </>
              ) : (
                <span className="text-[11px] font-semibold">Off</span>
              )}
            </button>
          );
        })}
      </div>
      <p className="text-xs font-semibold text-slate-500">
        Green = available every week · Gray = off · Tap a day to flip it
      </p>
    </div>
  );
}
