#!/usr/bin/env node

import { performance } from "node:perf_hooks";

const API_URL = process.env.API_GATEWAY_URL ?? "http://localhost:8081";
const LOCATION_NEAREST_HOSPITAL_URL = process.env.LOCATION_NEAREST_HOSPITAL_URL ?? "";
const PATIENT_PHONE = process.env.DEMO_PATIENT_PHONE ?? "+15555550100";
const PATIENT_EMAIL = process.env.DEMO_PATIENT_EMAIL ?? "";
const PATIENT_PASSWORD = process.env.DEMO_PATIENT_PASSWORD ?? "demo-password";
const HOSPITAL_EMAIL = process.env.DEMO_HOSPITAL_EMAIL ?? "staff@zorbahealth.local";
const HOSPITAL_PASSWORD = process.env.DEMO_HOSPITAL_PASSWORD ?? "demo-password";
const DEMO_PATIENT_ID = process.env.DEMO_PATIENT_ID ?? "";
const CONSENT_TYPE = process.env.DEMO_CONSENT_TYPE ?? "HEALTH_RECORD_ACCESS";
const QUESTION = process.env.DEMO_HEALTH_QUESTION ?? "What medications are in my records?";

const command = process.argv[2] ?? "all";

async function request(path, options = {}) {
  const startedAt = performance.now();
  const response = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(options.headers ?? {}),
    },
  });
  const latencyMs = Math.round(performance.now() - startedAt);
  const text = await response.text();
  let body = {};
  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      body = { raw: text };
    }
  }
  return { status: response.status, ok: response.ok, latencyMs, body };
}

function assertStep(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

function tokenFrom(result) {
  return result.body?.data?.access_token ?? result.body?.access_token ?? "";
}

function patientIDFrom(result) {
  return result.body?.data?.patient_id ?? result.body?.patient_id ?? DEMO_PATIENT_ID;
}

async function patientLogin() {
  const result = await request("/api/v1/auth/patient/login", {
    method: "POST",
    body: JSON.stringify({
      phone_number: PATIENT_PHONE,
      email: PATIENT_EMAIL,
      password: PATIENT_PASSWORD,
    }),
  });
  assertStep(result.ok, `patient login failed with HTTP ${result.status}`);
  const token = tokenFrom(result);
  assertStep(token, "patient login did not return an access token");
  return { result, token, patientID: patientIDFrom(result) };
}

async function hospitalLogin() {
  const result = await request("/api/v1/auth/hospital/login", {
    method: "POST",
    body: JSON.stringify({ email: HOSPITAL_EMAIL, password: HOSPITAL_PASSWORD }),
  });
  assertStep(result.ok, `hospital login failed with HTTP ${result.status}`);
  const token = tokenFrom(result);
  assertStep(token, "hospital login did not return an access token");
  return { result, token };
}

async function patientPortalSmoke() {
  const login = await patientLogin();
  const headers = { Authorization: `Bearer ${login.token}` };
  const checks = await Promise.all([
    request("/api/v1/patient/profile", { headers }),
    request("/api/v1/patient/consents", { headers }),
    request("/api/v1/patient/calls", { headers }),
    request("/api/v1/patient/audit", { headers }),
  ]);
  checks.forEach((result, index) => {
    assertStep(result.ok, `patient portal check ${index + 1} failed with HTTP ${result.status}`);
  });
  return {
    patientID: login.patientID,
    latencyMs: login.result.latencyMs + checks.reduce((sum, result) => sum + result.latencyMs, 0),
    assertions: ["login returned token", "profile loaded", "consents loaded", "calls loaded", "audit loaded"],
  };
}

async function grantConsent(token) {
  return request("/api/v1/patient/consents", {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
    body: JSON.stringify({
      consent_type: CONSENT_TYPE,
      scope: "patient-records",
      source: "evaluation-smoke",
    }),
  });
}

async function revokeConsent(token) {
  return request("/api/v1/patient/consents", {
    method: "DELETE",
    headers: { Authorization: `Bearer ${token}` },
    body: JSON.stringify({
      consent_type: CONSENT_TYPE,
      scope: "patient-records",
      source: "evaluation-smoke",
    }),
  });
}

async function recordsAnswer(token) {
  return request("/api/v1/patient/records/answer", {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
    body: JSON.stringify({ question: QUESTION, top_k: 5 }),
  });
}

async function consentGatingCheck() {
  const login = await patientLogin();
  const revoke = await revokeConsent(login.token);
  assertStep(revoke.ok, `consent revoke failed with HTTP ${revoke.status}`);
  const denied = await recordsAnswer(login.token);
  assertStep(!denied.ok, "records answer unexpectedly succeeded after consent revoke");
  const grant = await grantConsent(login.token);
  assertStep(grant.ok, `consent grant failed with HTTP ${grant.status}`);
  return {
    patientID: login.patientID,
    latencyMs: login.result.latencyMs + revoke.latencyMs + denied.latencyMs + grant.latencyMs,
    assertions: ["consent revoked", "records answer denied", "consent restored"],
  };
}

async function ragGroundednessCheck() {
  const login = await patientLogin();
  const grant = await grantConsent(login.token);
  assertStep(grant.ok, `consent grant failed with HTTP ${grant.status}`);
  const answer = await recordsAnswer(login.token);
  assertStep(answer.ok, `records answer failed with HTTP ${answer.status}`);
  const citations = answer.body?.data?.citations ?? answer.body?.citations ?? [];
  assertStep(Array.isArray(citations) && citations.length > 0, "records answer did not include citations");
  return {
    patientID: login.patientID,
    latencyMs: login.result.latencyMs + grant.latencyMs + answer.latencyMs,
    assertions: ["records answer succeeded", "answer returned at least one citation"],
  };
}

async function hospitalEscalationSmoke() {
  const login = await hospitalLogin();
  const headers = { Authorization: `Bearer ${login.token}` };
  const incidents = await request("/api/v1/hospital/incidents", { headers });
  assertStep(incidents.ok, `hospital incidents failed with HTTP ${incidents.status}`);
  const patientID = DEMO_PATIENT_ID || incidents.body?.data?.incidents?.[0]?.patient_id;
  assertStep(patientID, "DEMO_PATIENT_ID is required when no seeded incident exists");
  const audit = await request(`/api/v1/hospital/patient/audit?patient_id=${encodeURIComponent(patientID)}`, { headers });
  assertStep(audit.ok, `hospital patient audit failed with HTTP ${audit.status}`);
  return {
    patientID,
    latencyMs: login.result.latencyMs + incidents.latencyMs + audit.latencyMs,
    assertions: ["hospital login returned token", "incident list loaded", "patient audit loaded"],
  };
}

async function nearestHospitalSmoke() {
  if (!LOCATION_NEAREST_HOSPITAL_URL) {
    return {
      skipped: true,
      reason: "Set LOCATION_NEAREST_HOSPITAL_URL to an HTTP wrapper for location-service FindNearestHospital.",
    };
  }
  const startedAt = performance.now();
  const response = await fetch(LOCATION_NEAREST_HOSPITAL_URL);
  const body = await response.json().catch(() => ({}));
  assertStep(response.ok, `nearest hospital failed with HTTP ${response.status}`);
  assertStep(body.name || body.hospital?.name, "nearest hospital response did not include a name");
  return {
    latencyMs: Math.round(performance.now() - startedAt),
    assertions: ["nearest hospital returned a named facility"],
  };
}

const commands = {
  "patient-portal-smoke": patientPortalSmoke,
  "consent-gating-check": consentGatingCheck,
  "rag-groundedness-check": ragGroundednessCheck,
  "hospital-escalation-smoke": hospitalEscalationSmoke,
  "nearest-hospital-smoke": nearestHospitalSmoke,
};

async function runOne(name) {
  const startedAt = new Date().toISOString();
  try {
    const data = await commands[name]();
    return { name, status: data.skipped ? "skipped" : "passed", startedAt, ...data };
  } catch (error) {
    return { name, status: "failed", startedAt, error: error.message };
  }
}

if (command !== "all" && !commands[command]) {
  console.error(`Unknown command "${command}". Valid commands: all, ${Object.keys(commands).join(", ")}`);
  process.exit(2);
}

const names = command === "all" ? Object.keys(commands) : [command];
const results = [];
for (const name of names) {
  results.push(await runOne(name));
}

console.log(JSON.stringify({ apiUrl: API_URL, results }, null, 2));
process.exit(results.some((result) => result.status === "failed") ? 1 : 0);
