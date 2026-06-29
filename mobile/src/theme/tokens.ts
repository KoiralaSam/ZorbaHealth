export const colors = {
  canvas: "#e8eefc",
  background: "#f5f7ff",
  surface: "#ffffff",
  surfaceMuted: "#eef2ff",
  primary: "#4f46e5",
  primaryDeep: "#111c3a",
  primaryTint: "#cbd5ff",
  primaryTintStrong: "#e5e9ff",
  accent: "#fb923c",
  text: "#0b1120",
  subtleText: "#475569",
  mutedText: "#64748b",
  placeholder: "#94a3b8",
  border: "#dce3f4",
  borderSoft: "#dbeafe",
  success: "#047857",
  successBg: "#ecfdf5",
  error: "#b91c1c",
  errorBg: "#fff8f8",
  errorBorder: "#fecaca",
} as const;

export const spacing = {
  xs: 4,
  sm: 8,
  md: 12,
  lg: 16,
  xl: 20,
  '2xl': 24,
  '3xl': 32,
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
