import React, { useCallback, useEffect, useMemo, useState } from "react";
import {
  Pressable,
  ScrollView,
  Text,
  TextInput,
  View,
} from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { colors } from "../theme/tokens";
import { styles } from "../theme/styles";

export type PickerPatient = {
  patient_id?: string;
  full_name?: string;
  email?: string;
  phone_number?: string;
};

function fuzzyScore(query: string, patient: PickerPatient): number {
  const q = query.trim().toLowerCase();
  if (!q) return 1;
  const name = (patient.full_name || "").toLowerCase();
  const email = (patient.email || "").toLowerCase();
  const id = (patient.patient_id || "").toLowerCase();
  if (id === q) return 100;
  if (email === q) return 95;
  if (name === q) return 92;
  let score = 0;
  if (name.startsWith(q)) score = Math.max(score, 88);
  else if (name.includes(q)) score = Math.max(score, 72);
  if (email.includes(q)) score = Math.max(score, 65);
  if (id.includes(q)) score = Math.max(score, 55);
  const tokens = q.split(/\s+/).filter(Boolean);
  if (tokens.length > 1 && tokens.every((t) => name.includes(t))) score = Math.max(score, 82);
  return score;
}

function fuzzyFilter(patients: PickerPatient[], query: string): PickerPatient[] {
  return patients
    .map((patient) => ({ patient, score: fuzzyScore(query, patient) }))
    .filter((row) => row.score > 0)
    .sort(
      (a, b) =>
        b.score - a.score ||
        (a.patient.full_name || "").localeCompare(b.patient.full_name || ""),
    )
    .slice(0, 12)
    .map((row) => row.patient);
}

type Props = {
  token: string;
  value: string;
  onChange: (patientId: string, patient?: PickerPatient) => void;
  label?: string;
  loadPatients: (query: string) => Promise<PickerPatient[]>;
};

export function PatientPicker({
  token,
  value,
  onChange,
  label = "Patient",
  loadPatients,
}: Props) {
  const [allPatients, setAllPatients] = useState<PickerPatient[]>([]);
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [selectedPatient, setSelectedPatient] = useState<PickerPatient | null>(null);
  const [loaded, setLoaded] = useState(false);

  const ensureLoaded = useCallback(async () => {
    if (!token) return;
    if (loaded && allPatients.length > 0) return;
    setLoading(true);
    try {
      const list = await loadPatients("");
      setAllPatients(list);
      setLoaded(true);
    } catch {
      setAllPatients([]);
    } finally {
      setLoading(false);
    }
  }, [allPatients.length, loadPatients, loaded, token]);

  useEffect(() => {
    if (!value) {
      setSelectedPatient(null);
      return;
    }
    const match = allPatients.find((p) => p.patient_id === value);
    if (match) setSelectedPatient(match);
  }, [value, allPatients]);

  // When typing, also refresh server-side matches for broader results.
  useEffect(() => {
    if (!open || !token) return;
    const handle = setTimeout(() => {
      void (async () => {
        try {
          const list = await loadPatients(query);
          if (list.length > 0) {
            setAllPatients((prev) => {
              const byId = new Map(prev.map((p) => [p.patient_id, p]));
              for (const p of list) {
                if (p.patient_id) byId.set(p.patient_id, p);
              }
              return Array.from(byId.values());
            });
          }
        } catch {
          // keep existing list
        }
      })();
    }, 220);
    return () => clearTimeout(handle);
  }, [open, query, loadPatients, token]);

  const filtered = useMemo(() => fuzzyFilter(allPatients, query), [allPatients, query]);
  const shown = selectedPatient || allPatients.find((p) => p.patient_id === value);

  if (value && !open) {
    return (
      <View style={{ gap: 8 }}>
        <Text style={styles.label}>{label}</Text>
        <Pressable
          onPress={() => {
            onChange("");
            setSelectedPatient(null);
            setQuery("");
            setOpen(true);
            void ensureLoaded();
          }}
          style={[styles.card, { marginBottom: 0 }]}
        >
          <Text style={styles.cardTitle}>
            {shown?.full_name || `Patient ${value.slice(0, 8)}…`}
          </Text>
          <Text style={styles.meta}>{shown?.email || value}</Text>
          <Text style={[styles.meta, { color: colors.primary, marginTop: 6 }]}>
            Tap to change
          </Text>
        </Pressable>
      </View>
    );
  }

  return (
    <View style={{ gap: 8, zIndex: 20 }}>
      <Text style={styles.label}>{label}</Text>
      <View style={{ position: "relative" }}>
        <TextInput
          style={styles.input}
          value={query}
          onChangeText={(text) => {
            setQuery(text);
            setOpen(true);
          }}
          onFocus={() => {
            setOpen(true);
            void ensureLoaded();
          }}
          onBlur={() => {
            // Delay so option taps register first.
            setTimeout(() => setOpen(false), 180);
          }}
          placeholder="Search patient by name…"
          placeholderTextColor={colors.placeholder}
          autoCapitalize="none"
          autoCorrect={false}
        />
        {open ? (
          <View
            style={{
              marginTop: 6,
              maxHeight: 220,
              borderRadius: 16,
              borderWidth: 1,
              borderColor: colors.border,
              backgroundColor: colors.surface,
              overflow: "hidden",
            }}
          >
            {loading && allPatients.length === 0 ? (
              <Text style={[styles.meta, { padding: 12 }]}>Loading…</Text>
            ) : filtered.length === 0 ? (
              <Text style={[styles.meta, { padding: 12 }]}>
                {query.trim() ? "No patients matched." : "Start typing a name…"}
              </Text>
            ) : (
              <ScrollView keyboardShouldPersistTaps="handled" nestedScrollEnabled>
                {filtered.map((patient) => (
                  <Pressable
                    key={patient.patient_id}
                    onPress={() => {
                      setSelectedPatient(patient);
                      onChange(patient.patient_id || "", patient);
                      setOpen(false);
                      setQuery("");
                    }}
                    style={{
                      paddingHorizontal: 12,
                      paddingVertical: 10,
                      borderBottomWidth: 1,
                      borderBottomColor: colors.borderSoft,
                      flexDirection: "row",
                      alignItems: "center",
                      gap: 8,
                    }}
                  >
                    <View style={styles.flex}>
                      <Text style={styles.cardTitle}>
                        {patient.full_name || "Unnamed patient"}
                      </Text>
                      <Text style={styles.meta}>{patient.email || "No email"}</Text>
                    </View>
                    <Ionicons name="chevron-forward" size={16} color={colors.mutedText} />
                  </Pressable>
                ))}
              </ScrollView>
            )}
          </View>
        ) : null}
      </View>
    </View>
  );
}
