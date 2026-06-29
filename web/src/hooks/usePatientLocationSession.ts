"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { APIEndpoints } from "../contracts";
import { API_URL, LOCATION_WS_URL } from "../constants";

type WSCommand = {
  type?: string;
  data?: { session_id?: string; sessionID?: string } | string;
  session_id?: string;
  sessionID?: string;
};

type ConsentRow = { consent_type?: string; status?: string };

const GEO_OPTIONS: PositionOptions = {
  enableHighAccuracy: true,
  maximumAge: 60_000,
  timeout: 25_000,
};

const WATCH_OPTIONS: PositionOptions = {
  enableHighAccuracy: true,
  maximumAge: 5_000,
  timeout: 15_000,
};

export type PatientLocationSessionControls = {
  locationPermissionBlocked: boolean;
  retryBrowserLocation: () => void;
  /** LiveKit room SID for the current verified voice call (from location WS). */
  activeVoiceSessionId: string | null;
};

function sessionIDFromCommand(msg: WSCommand): string {
  const data = msg.data;
  if (typeof data === "object" && data !== null) {
    const fromData =
      (data as { session_id?: string; sessionID?: string }).session_id ??
      (data as { session_id?: string; sessionID?: string }).sessionID;
    if (fromData) {
      return String(fromData);
    }
  }
  if (msg.session_id) {
    return String(msg.session_id);
  }
  if (msg.sessionID) {
    return String(msg.sessionID);
  }
  return "";
}

async function fetchPatientConsents(accessToken: string): Promise<ConsentRow[]> {
  const controller = new AbortController();
  const timeout = window.setTimeout(() => controller.abort(), 4000);
  try {
    const res = await fetch(`${API_URL}${APIEndpoints.PATIENT_CONSENTS}`, {
      headers: { Authorization: `Bearer ${accessToken}` },
      signal: controller.signal,
    });
    if (!res.ok) {
      return [];
    }
    const body = (await res.json()) as {
      data?: { consents?: ConsentRow[] };
    };
    return body.data?.consents ?? [];
  } catch {
    return [];
  } finally {
    window.clearTimeout(timeout);
  }
}

export function usePatientLocationSession(
  accessToken: string | null,
  consents: ConsentRow[],
  onConsentsUpdated?: (consents: ConsentRow[]) => void,
  onLocationNotice?: (message: string) => void,
): PatientLocationSessionControls {
  const watchIdRef = useRef<number | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const activeVoiceSessionRef = useRef<string | null>(null);
  const accessTokenRef = useRef(accessToken);
  const consentsRef = useRef(consents);
  const onConsentsUpdatedRef = useRef(onConsentsUpdated);
  const onLocationNoticeRef = useRef(onLocationNotice);
  const [locationPermissionBlocked, setLocationPermissionBlocked] =
    useState(false);
  const [activeVoiceSessionId, setActiveVoiceSessionId] = useState<
    string | null
  >(null);

  accessTokenRef.current = accessToken;
  consentsRef.current = consents;
  onConsentsUpdatedRef.current = onConsentsUpdated;
  onLocationNoticeRef.current = onLocationNotice;

  const stopWatch = () => {
    if (watchIdRef.current != null && navigator.geolocation) {
      navigator.geolocation.clearWatch(watchIdRef.current);
      watchIdRef.current = null;
    }
  };

  const sendLocationUpdate = useCallback(
    (sessionID: string, lat: number, lng: number, accuracy: number) => {
      const ws = wsRef.current;
      if (!ws || ws.readyState !== WebSocket.OPEN) {
        console.warn("[location] skipped location_update: WebSocket not open", {
          sessionID,
        });
        return false;
      }
      ws.send(
        JSON.stringify({
          type: "location_update",
          sessionID,
          lat,
          lng,
          accuracy,
          method: "gps",
        }),
      );
      console.info("[location] sent location_update", {
        sessionID,
        method: "gps",
      });
      return true;
    },
    [],
  );

  const promptBrowserLocation = useCallback((sessionID: string, token: string) => {
    activeVoiceSessionRef.current = sessionID;
    setActiveVoiceSessionId(sessionID);

    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      console.info(
        "[location] deferring geolocation until WebSocket is open",
        sessionID,
      );
      return;
    }

    void fetchPatientConsents(token).then((fresh) => {
      if (fresh.length > 0) {
        consentsRef.current = fresh;
        onConsentsUpdatedRef.current?.(fresh);
      }
    });

    stopWatch();
    setLocationPermissionBlocked(false);
    onLocationNoticeRef.current?.("");

    if (!navigator.geolocation) {
      onLocationNoticeRef.current?.(
        "This browser does not support location sharing.",
      );
      return;
    }

    if (!window.isSecureContext) {
      onLocationNoticeRef.current?.(
        "Location requires HTTPS or localhost (e.g. http://localhost:3000).",
      );
      return;
    }

    console.info(
      "[location] showing browser location prompt for session",
      sessionID,
    );

    navigator.geolocation.getCurrentPosition(
      (pos) => {
        setLocationPermissionBlocked(false);
        onLocationNoticeRef.current?.("");
        sendLocationUpdate(
          sessionID,
          pos.coords.latitude,
          pos.coords.longitude,
          pos.coords.accuracy,
        );
        watchIdRef.current = navigator.geolocation.watchPosition(
          (update) => {
            sendLocationUpdate(
              sessionID,
              update.coords.latitude,
              update.coords.longitude,
              update.coords.accuracy,
            );
          },
          () => {},
          WATCH_OPTIONS,
        );
      },
      (err: GeolocationPositionError) => {
        if (err.code === 1) {
          setLocationPermissionBlocked(true);
          onLocationNoticeRef.current?.(
            "Location was blocked. Allow location for this site in the browser, or tap Try again below.",
          );
        } else if (err.code === 3) {
          onLocationNoticeRef.current?.("Location timed out. Try again.");
          setLocationPermissionBlocked(true);
        } else {
          onLocationNoticeRef.current?.("Could not read location. Try again.");
          setLocationPermissionBlocked(true);
        }
        console.warn("[location] geolocation error", err);
      },
      GEO_OPTIONS,
    );
  }, [sendLocationUpdate]);

  const promptBrowserLocationRef = useRef(promptBrowserLocation);
  promptBrowserLocationRef.current = promptBrowserLocation;

  const retryBrowserLocation = useCallback(() => {
    const token = accessTokenRef.current;
    const sessionID = activeVoiceSessionRef.current;
    if (!token || !sessionID) {
      return;
    }
    promptBrowserLocation(sessionID, token);
  }, [promptBrowserLocation]);

  useEffect(() => {
    if (!accessToken) {
      return;
    }

    const wsUrl = `${LOCATION_WS_URL}/ws/location?token=${encodeURIComponent(accessToken)}`;
    const ws = new WebSocket(wsUrl);
    wsRef.current = ws;

    const runPromptForActiveSession = () => {
      const sessionID = activeVoiceSessionRef.current;
      const token = accessTokenRef.current;
      if (sessionID && token) {
        promptBrowserLocationRef.current(sessionID, token);
      }
    };

    ws.onopen = () => {
      console.info("[location] WebSocket connected", LOCATION_WS_URL);
      runPromptForActiveSession();
    };

    ws.onmessage = (ev) => {
      let msg: WSCommand;
      try {
        msg = JSON.parse(String(ev.data)) as WSCommand;
      } catch {
        return;
      }
      const cmd = msg.type ?? "";
      const sessionID = sessionIDFromCommand(msg);
      if (cmd === "start_location" && sessionID) {
        console.info("[location] received start_location", sessionID);
        activeVoiceSessionRef.current = sessionID;
        setActiveVoiceSessionId(sessionID);
        const token = accessTokenRef.current;
        if (token) {
          promptBrowserLocationRef.current(sessionID, token);
        }
      } else if (cmd === "start_location") {
        console.warn("[location] start_location missing session_id", msg);
      }
      if (cmd === "stop_location") {
        activeVoiceSessionRef.current = null;
        setActiveVoiceSessionId(null);
        setLocationPermissionBlocked(false);
        onLocationNoticeRef.current?.("");
        stopWatch();
      }
    };

    ws.onerror = () => {
      console.warn("[location] WebSocket error", wsUrl);
    };

    ws.onclose = () => {
      stopWatch();
      wsRef.current = null;
    };

    return () => {
      stopWatch();
      ws.close();
      wsRef.current = null;
    };
  }, [accessToken]);

  return {
    locationPermissionBlocked,
    retryBrowserLocation,
    activeVoiceSessionId,
  };
}
