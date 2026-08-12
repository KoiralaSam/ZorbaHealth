import { Ionicons } from "@expo/vector-icons";
import DateTimePicker, {
  type DateTimePickerEvent,
} from "@react-native-community/datetimepicker";
import React, { useEffect, useMemo, useState } from "react";
import { Platform, Pressable, Text, View } from "react-native";
import { colors } from "../theme/tokens";
import { styles } from "../theme/styles";
import { resolveIANATimezone } from "../timezone";

function defaultScheduleDate() {
  const next = new Date();
  next.setMinutes(0, 0, 0);
  next.setHours(next.getHours() + 1);
  return next;
}

function mergeDatePart(base: Date, picked: Date) {
  const next = new Date(base);
  next.setFullYear(picked.getFullYear(), picked.getMonth(), picked.getDate());
  return next;
}

function mergeTimePart(base: Date, picked: Date) {
  const next = new Date(base);
  next.setHours(picked.getHours(), picked.getMinutes(), 0, 0);
  return next;
}

function formatScheduleLabel(value: Date, timeZone: string) {
  try {
    return value.toLocaleString(undefined, {
      timeZone,
      weekday: "short",
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "numeric",
      minute: "2-digit",
    });
  } catch {
    return value.toLocaleString(undefined, {
      weekday: "short",
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "numeric",
      minute: "2-digit",
    });
  }
}

/**
 * Native date + time chips. Always resolves and reports a Go-valid IANA timezone
 * (never GMT-05:00 / offset forms from Android Intl).
 */
export function ScheduleDateTimePicker({
  value,
  onChange,
  onTimezoneChange,
  label = "When",
}: {
  value: string;
  onChange: (isoValue: string) => void;
  onTimezoneChange?: (ianaTimezone: string) => void;
  label?: string;
}) {
  const timezone = useMemo(() => resolveIANATimezone(), []);
  const selected = useMemo(() => {
    if (!value.trim()) return defaultScheduleDate();
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? defaultScheduleDate() : parsed;
  }, [value]);

  const [showDate, setShowDate] = useState(false);
  const [showTime, setShowTime] = useState(false);

  useEffect(() => {
    onTimezoneChange?.(timezone);
  }, [onTimezoneChange, timezone]);

  const commit = (next: Date) => {
    onChange(next.toISOString());
    onTimezoneChange?.(timezone);
  };

  const onDateChange = (event: DateTimePickerEvent, date?: Date) => {
    if (Platform.OS === "android") {
      setShowDate(false);
    }
    if (event.type === "dismissed" || !date) {
      return;
    }
    commit(mergeDatePart(selected, date));
  };

  const onTimeChange = (event: DateTimePickerEvent, date?: Date) => {
    if (Platform.OS === "android") {
      setShowTime(false);
    }
    if (event.type === "dismissed" || !date) {
      return;
    }
    commit(mergeTimePart(selected, date));
  };

  return (
    <View style={styles.field}>
      <Text style={styles.label}>{label}</Text>
      <View style={styles.dateTimeRow}>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Choose date"
          onPress={() => {
            setShowTime(false);
            setShowDate(true);
            if (!value.trim()) commit(selected);
          }}
          style={({ pressed }) => [
            styles.dateTimeChip,
            pressed ? { opacity: 0.92 } : null,
          ]}
        >
          <Ionicons name="calendar-outline" size={18} color={colors.primary} />
          <View style={styles.flex}>
            <Text style={styles.dateTimeChipEyebrow}>Date</Text>
            <Text style={styles.dateTimeChipValue}>
              {selected.toLocaleDateString(undefined, {
                timeZone: timezone,
                weekday: "short",
                month: "short",
                day: "numeric",
                year: "numeric",
              })}
            </Text>
          </View>
        </Pressable>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel="Choose time"
          onPress={() => {
            setShowDate(false);
            setShowTime(true);
            if (!value.trim()) commit(selected);
          }}
          style={({ pressed }) => [
            styles.dateTimeChip,
            pressed ? { opacity: 0.92 } : null,
          ]}
        >
          <Ionicons name="time-outline" size={18} color={colors.primary} />
          <View style={styles.flex}>
            <Text style={styles.dateTimeChipEyebrow}>Time</Text>
            <Text style={styles.dateTimeChipValue}>
              {selected.toLocaleTimeString(undefined, {
                timeZone: timezone,
                hour: "numeric",
                minute: "2-digit",
              })}
            </Text>
          </View>
        </Pressable>
      </View>
      <Text style={styles.meta}>
        {formatScheduleLabel(selected, timezone)} · {timezone}
      </Text>

      {showDate ? (
        <DateTimePicker
          value={selected}
          mode="date"
          display={Platform.OS === "android" ? "calendar" : "spinner"}
          onChange={onDateChange}
          minimumDate={new Date()}
        />
      ) : null}
      {showTime ? (
        <DateTimePicker
          value={selected}
          mode="time"
          display={Platform.OS === "android" ? "clock" : "spinner"}
          onChange={onTimeChange}
          minuteInterval={1}
          is24Hour={false}
        />
      ) : null}
    </View>
  );
}
