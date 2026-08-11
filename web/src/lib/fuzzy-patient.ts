export type PatientSearchable = {
  patient_id?: string;
  full_name?: string;
  email?: string;
  phone_number?: string;
};

/** Score a patient against a free-text query. Higher is better; 0 means no match. */
export function fuzzyPatientScore(query: string, patient: PatientSearchable): number {
  const q = query.trim().toLowerCase();
  if (!q) return 1;

  const name = (patient.full_name || "").toLowerCase();
  const email = (patient.email || "").toLowerCase();
  const phone = (patient.phone_number || "").toLowerCase();
  const id = (patient.patient_id || "").toLowerCase();

  if (id === q) return 100;
  if (email === q) return 95;
  if (name === q) return 92;

  let score = 0;
  if (name.startsWith(q)) score = Math.max(score, 88);
  else if (name.includes(q)) score = Math.max(score, 72);

  if (email.startsWith(q)) score = Math.max(score, 80);
  else if (email.includes(q)) score = Math.max(score, 65);

  if (id.startsWith(q)) score = Math.max(score, 78);
  else if (id.includes(q)) score = Math.max(score, 55);

  if (phone.includes(q.replace(/[\s()-]/g, ""))) score = Math.max(score, 60);

  const tokens = q.split(/\s+/).filter(Boolean);
  if (tokens.length > 1 && tokens.every((t) => name.includes(t))) {
    score = Math.max(score, 82);
  }

  // Subsequence match on name (e.g. "jsm" → "Jane Smith")
  if (score === 0 && name && isSubsequence(q.replace(/\s+/g, ""), name.replace(/\s+/g, ""))) {
    score = 40;
  }

  return score;
}

function isSubsequence(needle: string, haystack: string): boolean {
  let ni = 0;
  for (let hi = 0; hi < haystack.length && ni < needle.length; hi++) {
    if (haystack[hi] === needle[ni]) ni++;
  }
  return ni === needle.length;
}

export function fuzzyFilterPatients<T extends PatientSearchable>(
  patients: T[],
  query: string,
  limit = 40,
): T[] {
  const q = query.trim();
  const scored = patients
    .map((patient) => ({ patient, score: fuzzyPatientScore(q, patient) }))
    .filter((row) => row.score > 0)
    .sort((a, b) => {
      if (b.score !== a.score) return b.score - a.score;
      return (a.patient.full_name || "").localeCompare(b.patient.full_name || "");
    });
  return scored.slice(0, limit).map((row) => row.patient);
}
