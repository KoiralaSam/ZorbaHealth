const MIN_VALID_DATE_YEAR = 2000;

export function meaningfulDate(value?: string) {
  if (!value) return null;
  const date = new Date(value);
  if (Number.isNaN(date.getTime()) || date.getFullYear() < MIN_VALID_DATE_YEAR) {
    return null;
  }
  return date;
}

export function formatDateTime(value?: string, fallback = "Unknown") {
  return meaningfulDate(value)?.toLocaleString() ?? fallback;
}

export function formatTimeOnly(value?: string, fallback = "Recently") {
  return (
    meaningfulDate(value)?.toLocaleTimeString([], {
      hour: "2-digit",
      minute: "2-digit",
    }) ?? fallback
  );
}

export function formatEventType(value?: string) {
  return value?.replaceAll("_", " ") || "Audit Event";
}
