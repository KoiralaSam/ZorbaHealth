import React from "react";
import { Pressable, Text, View } from "react-native";
import { colors } from "../theme/tokens";
import { styles } from "../theme/styles";

const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

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
    <View style={{ gap: 10 }}>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
        {WEEKDAYS.map((label, weekday) => {
          const on = active.has(weekday);
          return (
            <Pressable
              key={label}
              onPress={() => onToggle(weekday)}
              style={{
                width: "13%",
                minWidth: 44,
                flexGrow: 1,
                borderRadius: 16,
                paddingVertical: 12,
                paddingHorizontal: 4,
                alignItems: "center",
                backgroundColor: on ? colors.successBg : colors.surface,
                borderWidth: 1,
                borderColor: on ? colors.success : colors.border,
              }}
            >
              <Text
                style={{
                  fontWeight: "900",
                  fontSize: 12,
                  color: on ? colors.success : colors.mutedText,
                }}
              >
                {label}
              </Text>
              <Text
                style={{
                  marginTop: 6,
                  fontSize: 10,
                  fontWeight: "700",
                  textAlign: "center",
                  color: on ? colors.success : colors.placeholder,
                }}
              >
                {on ? `${startTime}\n${endTime}` : "Off"}
              </Text>
            </Pressable>
          );
        })}
      </View>
      <Text style={styles.meta}>Green = available every week · Gray = off · Tap to flip</Text>
    </View>
  );
}
