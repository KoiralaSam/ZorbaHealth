/**
 * Resolve a timezone string Go's time.LoadLocation will accept.
 * Android/Hermes Intl often returns offset forms (GMT-05:00) that fail IANA checks.
 */
export function isValidIANATimezone(tz: string): boolean {
  const value = tz.trim();
  if (!value) return false;
  // Offset / abbreviation forms are not IANA city zones and fail on the API.
  if (/^(GMT|UTC)?[+-]\d/i.test(value)) return false;
  if (/^[+-]\d{1,2}(:\d{2})?$/.test(value)) return false;
  try {
    Intl.DateTimeFormat("en-US", { timeZone: value }).format(new Date());
  } catch {
    return false;
  }
  // Prefer Region/City (America/Chicago) or the UTC/GMT aliases Go accepts.
  if (value === "UTC" || value === "GMT" || value.includes("/")) return true;
  return false;
}

export function resolveIANATimezone(preferred?: string | null): string {
  const candidates = [
    preferred?.trim(),
    (() => {
      try {
        return Intl.DateTimeFormat().resolvedOptions().timeZone;
      } catch {
        return undefined;
      }
    })(),
    "UTC",
  ];
  for (const candidate of candidates) {
    if (candidate && isValidIANATimezone(candidate)) {
      return candidate;
    }
  }
  return "UTC";
}
