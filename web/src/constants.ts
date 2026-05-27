// API Gateway URL
export const API_URL =
  process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8081";

/** Patient live location WebSocket (location-service, not API gateway). */
export const LOCATION_WS_URL =
  process.env.NEXT_PUBLIC_LOCATION_WS_URL ?? "ws://localhost:8091";

/** HTTP base for location-service (IP fallback only when GPS fails). */
export const LOCATION_HTTP_URL =
  process.env.NEXT_PUBLIC_LOCATION_HTTP_URL ??
  (process.env.NEXT_PUBLIC_LOCATION_WS_URL ?? "ws://localhost:8091").replace(
    /^ws/i,
    "http",
  );

// API Version
export const API_VERSION = "v1";
