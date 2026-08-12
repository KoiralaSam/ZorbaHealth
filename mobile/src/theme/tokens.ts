export const colors = {
  canvas: "#f1f5f9",
  background: "#f8fafc",
  surface: "#ffffff",
  surfaceMuted: "#f1f5f9",
  primary: "#4338ca",
  primaryDeep: "#0f172a",
  primaryTint: "#e0e7ff",
  primaryTintStrong: "#eef2ff",
  accent: "#b45309",
  text: "#0f172a",
  subtleText: "#475569",
  mutedText: "#64748b",
  placeholder: "#94a3b8",
  border: "#cbd5e1",
  borderSoft: "#dbeafe",
  success: "#047857",
  successBg: "#ecfdf5",
  error: "#b91c1c",
  errorBg: "#fff8f8",
  errorBorder: "#fecaca",
  info: "#1d4ed8",
  infoBg: "#eff6ff",
  caution: "#92400e",
  cautionBg: "#fffbeb",
  phi: "#5b21b6",
  phiBg: "#f5f3ff",
} as const;

export const spacing = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 20,
  '2xl': 24,
  '3xl': 32,
  '4xl': 40,
} as const;

export const radii = {
  sm: 12,
  md: 16,
  lg: 20,
  xl: 24,
  pill: 999,
} as const;

export const typeScale = {
  xs: 11,
  sm: 13,
  md: 15,
  lg: 18,
  xl: 24,
  hero: 28,
} as const;

export const shadows = {
  soft: {
    shadowColor: colors.text,
    shadowOpacity: 0.05,
    shadowRadius: 16,
    shadowOffset: { width: 0, height: 7 },
    elevation: 3,
  },
  glow: {
    shadowColor: colors.primary,
    shadowOpacity: 0.18,
    shadowRadius: 18,
    shadowOffset: { width: 0, height: 8 },
    elevation: 5,
  },
} as const;
