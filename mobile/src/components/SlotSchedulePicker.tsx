import React, { useEffect, useMemo, useState } from "react";
import { Pressable, Text, View } from "react-native";
import { colors } from "../theme/tokens";
import { styles } from "../theme/styles";

export type AppointmentSlotOption = {
  starts_at: string;
  ends_at: string;
  duration_minutes: number;
  timezone?: string;
};

type Props = {
  slots: AppointmentSlotOption[];
  value: string;
  onChange: (startsAt: string) => void;
  loading?: boolean;
  emptyText?: string;
  /** When true, calendar stays visible but greyed and non-interactive. */
  disabled?: boolean;
  disabledText?: string;
  /** Optional controlled selected date YYYY-MM-DD */
  selectedDate?: string;
  onSelectedDateChange?: (isoDate: string) => void;
};

const WEEKDAYS = ["S", "M", "T", "W", "T", "F", "S"];

function toISODate(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function formatTime(iso: string): string {
  try {
    return new Date(iso).toLocaleTimeString(undefined, {
      hour: "numeric",
      minute: "2-digit",
    });
  } catch {
    return iso;
  }
}

/** Month calendar: green dates are bookable; tap auto-selects earliest slot that day. */
export function SlotSchedulePicker({
  slots,
  value,
  onChange,
  loading,
  emptyText = "No openings in the next 4 weeks. Ask your care team to set availability.",
  disabled = false,
  disabledText = "Select a hospital and doctor to see open dates.",
  selectedDate: controlledDate,
  onSelectedDateChange,
}: Props) {
  const availableDates = useMemo(() => {
    const set = new Set<string>();
    for (const slot of slots) set.add(slot.starts_at.slice(0, 10));
    return Array.from(set).sort();
  }, [slots]);

  const [cursor, setCursor] = useState(() => {
    const seed = availableDates[0] ? new Date(`${availableDates[0]}T12:00:00`) : new Date();
    return new Date(seed.getFullYear(), seed.getMonth(), 1);
  });
  const [internalDate, setInternalDate] = useState(availableDates[0] ?? "");
  const selectedDate = controlledDate ?? internalDate;

  const setSelectedDate = (iso: string) => {
    if (disabled) return;
    if (onSelectedDateChange) onSelectedDateChange(iso);
    else setInternalDate(iso);
  };

  useEffect(() => {
    if (disabled) return;
    if (!selectedDate && availableDates.length > 0) {
      setSelectedDate(availableDates[0]);
    }
  }, [availableDates, selectedDate, disabled]);

  const slotsForDate = useMemo(() => {
    if (disabled || !selectedDate) return [];
    return slots.filter((s) => s.starts_at.slice(0, 10) === selectedDate);
  }, [slots, selectedDate, disabled]);

  useEffect(() => {
    if (disabled) return;
    if (slotsForDate.length === 0) {
      if (value) onChange("");
      return;
    }
    if (!slotsForDate.some((s) => s.starts_at === value)) {
      onChange(slotsForDate[0].starts_at);
    }
  }, [slotsForDate, value, onChange, disabled]);

  const cells = useMemo(() => {
    const first = new Date(cursor.getFullYear(), cursor.getMonth(), 1);
    const startPad = first.getDay();
    const daysInMonth = new Date(cursor.getFullYear(), cursor.getMonth() + 1, 0).getDate();
    const out: Array<{ iso: string; day: number } | null> = [];
    for (let i = 0; i < startPad; i++) out.push(null);
    for (let day = 1; day <= daysInMonth; day++) {
      const d = new Date(cursor.getFullYear(), cursor.getMonth(), day);
      out.push({ iso: toISODate(d), day });
    }
    while (out.length % 7 !== 0) out.push(null);
    return out;
  }, [cursor]);

  const availableSet = useMemo(() => new Set(availableDates), [availableDates]);
  const todayISO = toISODate(new Date());
  const monthLabel = cursor.toLocaleString(undefined, { month: "long", year: "numeric" });

  return (
    <View style={[styles.stack, disabled ? { opacity: 0.45 } : null]}>
      <View style={[styles.rowBetween, { marginBottom: 8 }]}>
        <Pressable
          disabled={disabled}
          onPress={() => setCursor(new Date(cursor.getFullYear(), cursor.getMonth() - 1, 1))}
          style={styles.chip}
        >
          <Text style={styles.meta}>‹</Text>
        </Pressable>
        <Text style={styles.cardTitle}>{monthLabel}</Text>
        <Pressable
          disabled={disabled}
          onPress={() => setCursor(new Date(cursor.getFullYear(), cursor.getMonth() + 1, 1))}
          style={styles.chip}
        >
          <Text style={styles.meta}>›</Text>
        </Pressable>
      </View>

      <View style={{ flexDirection: "row", marginBottom: 4 }}>
        {WEEKDAYS.map((d, i) => (
          <View key={`${d}-${i}`} style={{ flex: 1, alignItems: "center" }}>
            <Text style={[styles.meta, { fontSize: 11 }]}>{d}</Text>
          </View>
        ))}
      </View>

      <View style={{ flexDirection: "row", flexWrap: "wrap" }}>
        {cells.map((cell, idx) => {
          if (!cell) {
            return <View key={`pad-${idx}`} style={{ width: "14.28%", height: 40 }} />;
          }
          const isAvailable = !disabled && availableSet.has(cell.iso);
          const isPast = cell.iso < todayISO;
          const selectable = !disabled && isAvailable && !isPast;
          const selected = !disabled && selectedDate === cell.iso;
          return (
            <Pressable
              key={cell.iso}
              disabled={!selectable}
              onPress={() => setSelectedDate(cell.iso)}
              style={{
                width: "14.28%",
                height: 40,
                alignItems: "center",
                justifyContent: "center",
                borderRadius: 10,
                backgroundColor: selected
                  ? colors.text
                  : selectable
                    ? colors.successBg
                    : "transparent",
              }}
            >
              <Text
                style={{
                  fontWeight: "700",
                  color: selected
                    ? "#fff"
                    : selectable
                      ? colors.success
                      : colors.placeholder,
                }}
              >
                {cell.day}
              </Text>
            </Pressable>
          );
        })}
      </View>

      <Text style={[styles.meta, { marginTop: 12 }]}>
        {disabled
          ? disabledText
          : loading
            ? "Loading available times…"
            : `Times on ${selectedDate || "—"} (earliest auto-selected)`}
      </Text>

      {!disabled && !loading && availableDates.length === 0 ? (
        <Text style={styles.meta}>{emptyText}</Text>
      ) : null}

      {!disabled && !loading && selectedDate && slotsForDate.length === 0 && availableDates.length > 0 ? (
        <Text style={styles.meta}>No openings on this date.</Text>
      ) : null}

      {!disabled ? (
        <View style={styles.inlineActions}>
          {slotsForDate.map((s) => {
            const selected = value === s.starts_at;
            return (
              <Pressable
                key={s.starts_at}
                onPress={() => onChange(s.starts_at)}
                style={[
                  styles.chip,
                  selected ? { backgroundColor: colors.accent } : { backgroundColor: "#ecfdf5" },
                ]}
              >
                <Text
                  style={
                    selected
                      ? { color: "#fff", fontWeight: "700" }
                      : { color: "#065f46", fontWeight: "700" }
                  }
                >
                  {formatTime(s.starts_at)}
                </Text>
              </Pressable>
            );
          })}
        </View>
      ) : null}

      {!disabled && !loading && availableDates.length > 0 ? (
        <Text style={[styles.meta, { marginTop: 4 }]}>
          Green dates have open slots. Tap one to auto-select the earliest time.
        </Text>
      ) : null}
    </View>
  );
}
