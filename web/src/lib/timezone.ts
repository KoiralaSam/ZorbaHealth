/**
 * Resolve a timezone string Go's time.LoadLocation will accept.
 * Some browsers report offset forms (GMT-05:00) that fail IANA checks.
 */
export function isValidIANATimezone(tz: string): boolean {
  const value = tz.trim();
  if (!value) return false;
  if (/^(GMT|UTC)?[+-]\d/i.test(value)) return false;
  if (/^[+-]\d{1,2}(:\d{2})?$/.test(value)) return false;
  try {
    Intl.DateTimeFormat("en-US", { timeZone: value }).format(new Date());
  } catch {
    return false;
  }
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
