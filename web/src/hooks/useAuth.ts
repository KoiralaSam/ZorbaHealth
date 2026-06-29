"use client";

import { useCallback, useEffect, useState } from "react";
import {
  AuthRole,
  authStateMatchesRole,
  bootstrapAuth,
  getAuthState,
  setAuth,
  subscribeAuth,
} from "../lib/auth-client";

export function useAuth(role: AuthRole) {
  const [, tick] = useState(0);
  useEffect(() => {
    const unsubscribe = subscribeAuth(() => tick((n) => n + 1));
    return unsubscribe;
  }, []);
  const [ready, setReady] = useState(false);

  const boot = useCallback(async () => {
    await bootstrapAuth(role);
    setReady(true);
  }, [role]);

  useEffect(() => {
    void boot();
  }, [boot]);

  const auth = getAuthState();
  const authenticated = authStateMatchesRole(auth, role);

  return {
    ready,
    authenticated,
    accessToken: auth?.accessToken ?? "",
    patientId: auth?.patientId,
    hospitalId: auth?.hospitalId,
    staffId: auth?.staffId,
    staffRole: auth?.staffRole,
    setSession: (accessToken: string, patientId?: string) => {
      setAuth({ role, accessToken, patientId });
      setReady(true);
    },
    clearSession: () => {
      setAuth(null);
    },
  };
}
