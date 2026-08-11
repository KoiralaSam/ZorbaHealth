/**
 * Resolve a timezone string Go's time.LoadLocation will accept.
 * Some browsers/devices report offset forms (GMT-05:00) that fail IANA checks.
 */

const COMMON_IANA_TIMEZONES = [
  "UTC",
  "America/New_York",
  "America/Chicago",
  "America/Denver",
  "America/Los_Angeles",
  "America/Phoenix",
  "America/Anchorage",
  "America/Honolulu",
  "America/Toronto",
  "America/Vancouver",
  "America/Mexico_City",
  "America/Sao_Paulo",
  "Europe/London",
  "Europe/Paris",
  "Europe/Berlin",
  "Europe/Madrid",
  "Europe/Rome",
  "Europe/Amsterdam",
  "Europe/Moscow",
  "Asia/Kolkata",
  "Asia/Dubai",
  "Asia/Singapore",
  "Asia/Shanghai",
  "Asia/Tokyo",
  "Asia/Seoul",
  "Australia/Sydney",
  "Pacific/Auckland",
] as const;

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

function offsetMinutesAt(date: Date, timeZone: string): number | null {
  try {
    const parts = new Intl.DateTimeFormat("en-US", {
      timeZone,
      timeZoneName: "shortOffset",
      hour: "2-digit",
      minute: "2-digit",
      hourCycle: "h23",
    }).formatToParts(date);
    const raw = parts.find((part) => part.type === "timeZoneName")?.value ?? "";
    if (raw === "GMT" || raw === "UTC") return 0;
    const match = raw.match(/([+-])(\d{1,2})(?::?(\d{2}))?/);
    if (!match) return null;
    const sign = match[1] === "-" ? -1 : 1;
    const hours = Number(match[2] || 0);
    const minutes = Number(match[3] || 0);
    return sign * (hours * 60 + minutes);
  } catch {
    return null;
  }
}

function guessIANAFromDeviceOffset(): string | undefined {
  const samples = [
    new Date(),
    new Date(Date.UTC(new Date().getUTCFullYear(), 0, 15, 12)),
    new Date(Date.UTC(new Date().getUTCFullYear(), 6, 15, 12)),
  ];
  const deviceOffsets = samples.map((date) => -date.getTimezoneOffset());

  for (const candidate of COMMON_IANA_TIMEZONES) {
    if (!isValidIANATimezone(candidate)) continue;
    const matches = samples.every((date, index) => {
      const zoneOffset = offsetMinutesAt(date, candidate);
      return zoneOffset !== null && zoneOffset === deviceOffsets[index];
    });
    if (matches) return candidate;
  }

  const nowOffset = deviceOffsets[0];
  for (const candidate of COMMON_IANA_TIMEZONES) {
    if (!isValidIANATimezone(candidate)) continue;
    const zoneOffset = offsetMinutesAt(samples[0], candidate);
    if (zoneOffset === nowOffset) return candidate;
  }
  return undefined;
}

export function resolveIANATimezone(preferred?: string | null): string {
  const deviceZone = (() => {
    try {
      return Intl.DateTimeFormat().resolvedOptions().timeZone;
    } catch {
      return undefined;
    }
  })();

  const candidates = [
    preferred?.trim(),
    deviceZone,
    guessIANAFromDeviceOffset(),
    "UTC",
  ];
  for (const candidate of candidates) {
    if (candidate && isValidIANATimezone(candidate)) {
      return candidate;
    }
  }
  return "UTC";
}
