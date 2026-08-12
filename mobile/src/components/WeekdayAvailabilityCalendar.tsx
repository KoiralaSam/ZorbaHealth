import React from "react";
import { WeeklyScheduleBoard } from "./WeeklyScheduleBoard";

type LegacyProps = {
  activeWeekdays: number[];
  onToggleWeekday?: (weekday: number, isoDate: string) => void;
  onToggle?: (weekday: number) => void;
  selectedDate?: string;
  onSelectDate?: (isoDate: string) => void;
  label?: string;
  startTime?: string;
  endTime?: string;
};

/**
 * Compatibility wrapper for older call sites / stale Metro bundles.
 * Prefer WeeklyScheduleBoard for new code.
 */
export function WeekdayAvailabilityCalendar({
  activeWeekdays,
  onToggleWeekday,
  onToggle,
  selectedDate = "",
  startTime = "09:00",
  endTime = "17:00",
}: LegacyProps) {
  return (
    <WeeklyScheduleBoard
      activeWeekdays={activeWeekdays}
      startTime={startTime}
      endTime={endTime}
      onToggle={(weekday) => {
        onToggle?.(weekday);
        onToggleWeekday?.(weekday, selectedDate);
      }}
    />
  );
}
