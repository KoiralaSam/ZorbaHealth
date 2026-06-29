import { API_URL } from "../constants";
import { APIEndpoints } from "../contracts";
import type { APIResponse } from "../types";

export type AuthRole = "patient" | "hospital" | "hospital_staff";

type AuthState = {
  role: AuthRole;
  accessToken: string;
  patientId?: string;
  hospitalId?: string;
  staffId?: string;
  staffRole?: string;
};

let state: AuthState | null = null;
let bootstrapPromise: Promise<AuthState | null> | null = null;
let bootstrapRole: AuthRole | null = null;
let refreshPromise: Promise<string | null> | null = null;

const listeners = new Set<() => void>();
const apiCache = new Map<
  string,
  {
    expiresAt: number;
    value?: unknown;
    promise?: Promise<unknown>;
  }
>();

type CachedApiOptions = {
  ttlMs?: number;
  force?: boolean;
};

function notify() {
  listeners.forEach((l) => l());
}

export function subscribeAuth(listener: () => void): () => void {
  listeners.add(listener);
  return () => {
    listeners.delete(listener);
  };
}

export function getAuthState() {
  return state;
}

export function setAuth(next: AuthState | null) {
  if (!next || next.accessToken !== state?.accessToken) {
    clearApiCache();
  }
  state = next;
  notify();
}

function isHospitalRole(role: AuthRole) {
  return role === "hospital" || role === "hospital_staff";
}

function refreshPathForRole(role: AuthRole) {
  return role === "patient"
    ? APIEndpoints.PATIENT_REFRESH
    : APIEndpoints.HOSPITAL_REFRESH;
}

export function authStateMatchesRole(auth: AuthState | null, role: AuthRole) {
  if (!auth?.accessToken) return false;
  if (role === "patient") return auth.role === "patient";
  return isHospitalRole(role) && isHospitalRole(auth.role);
}

async function postJSON<T>(path: string, body?: unknown): Promise<{ ok: boolean; status: number; payload: APIResponse<T> }> {
  const response = await fetch(`${API_URL}${path}`, {
    method: "POST",
    credentials: "include",
    headers: body === undefined ? undefined : { "Content-Type": "application/json" },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const payload = (await response.json().catch(() => ({}))) as APIResponse<T>;
  return { ok: response.ok, status: response.status, payload };
}

export async function bootstrapAuth(role: AuthRole): Promise<AuthState | null> {
  if (authStateMatchesRole(state, role)) return state;
  if (bootstrapPromise && bootstrapRole === role) return bootstrapPromise;
  bootstrapRole = role;
  bootstrapPromise = (async () => {
    const res = await postJSON<{
      access_token?: string;
      patient_id?: string;
      hospital_id?: string;
      staff_id?: string;
      role?: string;
    }>(refreshPathForRole(role));
    if (!res.ok) {
      setAuth(null);
      return null;
    }
    const token = res.payload.data?.access_token;
    if (!token) {
      setAuth(null);
      return null;
    }
    const next: AuthState = { role, accessToken: token };
    if (role === "patient" && res.payload.data?.patient_id) {
      next.patientId = res.payload.data.patient_id;
    }
    if (isHospitalRole(role)) {
      next.hospitalId = res.payload.data?.hospital_id;
      next.staffId = res.payload.data?.staff_id;
      next.staffRole = res.payload.data?.role;
    }
    setAuth(next);
    return next;
  })();
  return bootstrapPromise;
}

export async function refreshAccessToken(role: AuthRole): Promise<string | null> {
  if (refreshPromise) return refreshPromise;
  refreshPromise = (async () => {
    const res = await postJSON<{
      access_token?: string;
      hospital_id?: string;
      staff_id?: string;
      role?: string;
    }>(refreshPathForRole(role));
    refreshPromise = null;
    if (!res.ok) {
      if (res.payload.error?.code === "REFRESH_TOKEN_REUSE") {
        setAuth(null);
      }
      return null;
    }
    const token = res.payload.data?.access_token;
    if (!token) return null;
    if (state) {
      setAuth({
        ...state,
        role,
        accessToken: token,
        hospitalId: res.payload.data?.hospital_id ?? state.hospitalId,
        staffId: res.payload.data?.staff_id ?? state.staffId,
        staffRole: res.payload.data?.role ?? state.staffRole,
      });
    }
    return token;
  })();
  return refreshPromise;
}

export async function apiFetch(
  role: AuthRole,
  path: string,
  init: RequestInit = {},
): Promise<Response> {
  const token = state?.accessToken ?? "";
  const headers = new Headers(init.headers);
  if (token) headers.set("Authorization", `Bearer ${token}`);
  const response = await fetch(`${API_URL}${path}`, {
    ...init,
    credentials: "include",
    headers,
  });
  if (response.status !== 401) return response;
  const refreshed = await refreshAccessToken(role);
  if (!refreshed) return response;
  const retryHeaders = new Headers(init.headers);
  retryHeaders.set("Authorization", `Bearer ${refreshed}`);
  return fetch(`${API_URL}${path}`, {
    ...init,
    credentials: "include",
    headers: retryHeaders,
  });
}

function cacheKey(role: AuthRole, path: string) {
  const token = state?.accessToken ?? "";
  return `${role}:${token.slice(-16)}:${path}`;
}

export function clearApiCache(prefix?: string) {
  if (!prefix) {
    apiCache.clear();
    return;
  }
  for (const key of apiCache.keys()) {
    if (key.startsWith(prefix)) apiCache.delete(key);
  }
}

export async function cachedApiJSON<T>(
  role: AuthRole,
  path: string,
  { ttlMs = 30_000, force = false }: CachedApiOptions = {},
): Promise<T> {
  const key = cacheKey(role, path);
  const now = Date.now();
  const cached = apiCache.get(key);
  if (!force && cached?.value !== undefined && cached.expiresAt > now) {
    return cached.value as T;
  }
  if (!force && cached?.promise) {
    return cached.promise as Promise<T>;
  }

  const promise = apiFetch(role, path).then(async (response) => {
    const payload = (await response.json().catch(() => ({}))) as APIResponse<T> | T;
    if (!response.ok) {
      const errorPayload = payload as APIResponse<T>;
      throw new Error(errorPayload.error?.message || "Request failed.");
    }
    apiCache.set(key, {
      expiresAt: Date.now() + ttlMs,
      value: payload,
    });
    return payload;
  });

  apiCache.set(key, { expiresAt: now + ttlMs, promise });
  try {
    return (await promise) as T;
  } catch (error) {
    apiCache.delete(key);
    throw error;
  }
}

export function preloadApiJSON<T>(
  role: AuthRole,
  path: string,
  options?: CachedApiOptions,
) {
  void cachedApiJSON<T>(role, path, options).catch(() => {
    /* preloading should never surface user-facing errors */
  });
}

export async function logoutAuth(role: AuthRole) {
  const path =
    role === "patient" ? APIEndpoints.PATIENT_LOGOUT : APIEndpoints.HOSPITAL_LOGOUT;
  await postJSON(path);
  clearApiCache();
  setAuth(null);
  bootstrapPromise = null;
  bootstrapRole = null;
}
