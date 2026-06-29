import { Ionicons } from "@expo/vector-icons";
import React from "react";
import {
  ActivityIndicator,
  Pressable,
  type PressableStateCallbackType,
  ScrollView,
  Text,
  TextInput,
  View,
} from "react-native";
import { colors } from "../theme/tokens";
import { styles } from "../theme/styles";

export function Field(props: {
  label: string;
  value: string;
  onChangeText: (value: string) => void;
  placeholder?: string;
  secureTextEntry?: boolean;
  keyboardType?: "default" | "email-address" | "number-pad" | "phone-pad";
  autoCapitalize?: "none" | "sentences" | "words" | "characters";
  multiline?: boolean;
}) {
  return (
    <View style={styles.field}>
      <Text style={styles.label}>{props.label}</Text>
      <TextInput
        style={[styles.input, props.multiline ? styles.textArea : null]}
        value={props.value}
        onChangeText={props.onChangeText}
        placeholder={props.placeholder}
        secureTextEntry={props.secureTextEntry}
        keyboardType={props.keyboardType}
        autoCapitalize={props.autoCapitalize}
        multiline={props.multiline}
        placeholderTextColor={colors.placeholder}
      />
    </View>
  );
}

export function PrimaryButton({
  icon,
  label,
  onPress,
  disabled = false,
  accessibilityLabel,
}: {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  disabled?: boolean;
  accessibilityLabel?: string;
}) {
  return (
    <Pressable
      disabled={disabled}
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel ?? label}
      style={({ pressed }: PressableStateCallbackType) => [
        styles.primaryButton,
        disabled ? styles.disabled : null,
        pressed && !disabled ? { opacity: 0.92, transform: [{ scale: 0.985 }] } : null,
      ]}
    >
      <Ionicons name={icon} size={18} color={colors.surface} />
      <Text style={styles.primaryText}>{label}</Text>
    </Pressable>
  );
}

export function IconButton({
  icon,
  label,
  onPress,
  tone,
  accessibilityLabel,
}: {
  icon: keyof typeof Ionicons.glyphMap;
  label: string;
  onPress: () => void;
  tone: "neutral" | "accent";
  accessibilityLabel?: string;
}) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel ?? label}
      style={({ pressed }: PressableStateCallbackType) => [
        styles.iconButton,
        tone === "accent" ? styles.iconAccent : null,
        pressed ? { opacity: 0.9, transform: [{ scale: 0.98 }] } : null,
      ]}
    >
      <Ionicons
        name={icon}
        size={16}
        color={tone === "accent" ? colors.surface : colors.subtleText}
      />
      <Text
        style={[
          styles.iconButtonText,
          tone === "accent" ? styles.iconAccentText : null,
        ]}
      >
        {label}
      </Text>
    </Pressable>
  );
}

export function TextButton({
  label,
  onPress,
  accessibilityLabel,
}: {
  label: string;
  onPress: () => void;
  accessibilityLabel?: string;
}) {
  return (
    <Pressable
      onPress={onPress}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel ?? label}
      style={({ pressed }: PressableStateCallbackType) => [
        styles.textButton,
        pressed ? { opacity: 0.75 } : null,
      ]}
    >
      <Text style={styles.textButtonLabel}>{label}</Text>
    </Pressable>
  );
}

export function Segmented({
  value,
  options,
  onChange,
}: {
  value: string;
  options: { value: string; label: string }[];
  onChange: (value: string) => void;
}) {
  return (
    <View style={styles.segmented}>
      {options.map((option) => {
        const active = value === option.value;
        return (
          <Pressable
            key={option.value}
            onPress={() => onChange(option.value)}
            accessibilityRole="button"
            accessibilityLabel={option.label}
            style={({ pressed }: PressableStateCallbackType) => [
              styles.segment,
              active ? styles.segmentActive : null,
              pressed ? { opacity: 0.85 } : null,
            ]}
          >
            <Text
              style={[
                styles.segmentText,
                active ? styles.segmentTextActive : null,
              ]}
            >
              {option.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

export function TabBar({
  value,
  options,
  onChange,
}: {
  value: string;
  options: {
    value: string;
    label: string;
    icon: keyof typeof Ionicons.glyphMap;
  }[];
  onChange: (value: string) => void;
}) {
  return (
    <View style={styles.tabBarFrame}>
      <ScrollView
        horizontal
        showsHorizontalScrollIndicator={false}
        bounces={false}
        contentContainerStyle={styles.tabBar}
      >
        {options.map((option) => {
          const active = value === option.value;

          return (
            <Pressable
              key={option.value}
              onPress={() => onChange(option.value)}
              accessibilityRole="tab"
              accessibilityLabel={option.label}
              accessibilityState={{ selected: active }}
              style={({ pressed }: PressableStateCallbackType) => [
                styles.tab,
                active && styles.tabActive,
                pressed ? { opacity: 0.9 } : null,
              ]}
            >
              <View style={styles.tabInner}>
                <Ionicons
                  name={option.icon}
                  size={18}
                  color={active ? "#fff" : "#475569"}
                />

                <Text
                  numberOfLines={1}
                  style={[styles.tabText, active && styles.tabTextActive]}
                >
                  {option.label}
                </Text>
              </View>
            </Pressable>
          );
        })}
      </ScrollView>
    </View>
  );
}

export function Section({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <View style={styles.section}>
      <Text style={styles.sectionTitle}>{title}</Text>
      <View style={styles.stack}>{children}</View>
    </View>
  );
}

export function ScreenHeading({
  title,
  subtitle,
}: {
  title: string;
  subtitle: string;
}) {
  return (
    <View style={styles.headingBlock}>
      <Text style={styles.screenTitle}>{title}</Text>
      <Text style={styles.screenCopy}>{subtitle}</Text>
    </View>
  );
}

export function InfoCard({
  icon,
  title,
  body,
}: {
  icon: keyof typeof Ionicons.glyphMap;
  title: string;
  body: string;
}) {
  return (
    <View style={styles.infoCard}>
      <View style={styles.infoCardHeader}>
        <Ionicons name={icon} size={18} color={colors.primary} />
        <Text style={styles.infoCardTitle}>{title}</Text>
      </View>
      <Text style={styles.infoCardBody}>{body}</Text>
    </View>
  );
}

export function LoadingCard() {
  return (
    <View style={styles.card}>
      <ActivityIndicator size="small" color={colors.primary} />
      <Text style={[styles.muted, { textAlign: "center", marginTop: 8 }]}>
        Loading security workspace...
      </Text>
    </View>
  );
}

export function EmptyText({ text }: { text: string }) {
  return <Text style={styles.emptyText}>{text}</Text>;
}

export function Feedback({ error, notice }: { error?: string; notice?: string }) {
  if (!error && !notice) return null;
  return (
    <View
      style={[
        styles.feedback,
        error ? styles.feedbackError : styles.feedbackNotice,
      ]}
    >
      <View style={{ flexDirection: "row", alignItems: "center", gap: 6 }}>
        <Ionicons
          name={error ? "alert-circle" : "checkmark-circle"}
          size={16}
          color={error ? colors.error : colors.primary}
        />
        <Text style={error ? styles.errorText : styles.noticeText}>
          {error || notice}
        </Text>
      </View>
    </View>
  );
}
