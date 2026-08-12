export const colors = {
  canvas: "#f1f5f9",
  background: "#f8fafc",
  surface: "#ffffff",
<<<<<<< HEAD
  surfaceMuted: "#f1f5f9",
  primary: "#4338ca",
  primaryDeep: "#0f172a",
  primaryTint: "#e0e7ff",
  primaryTintStrong: "#eef2ff",
  accent: "#b45309",
  text: "#0f172a",
=======
  surfaceMuted: "#eef2ff",
  primary: "#4f46e5",
  primaryDeep: "#111c3a",
  primaryTint: "#cbd5ff",
  primaryTintStrong: "#e5e9ff",
  secondary: "#0891b2",
  accent: "#fb923c",
  text: "#0b1120",
>>>>>>> af5074b (Sync active ZorbaHealth changes)
  subtleText: "#475569",
  mutedText: "#64748b",
  placeholder: "#94a3b8",
  border: "#cbd5e1",
  borderSoft: "#dbeafe",
  success: "#047857",
  successBg: "#ecfdf5",
  successBorder: "#a7f3d0",
  warning: "#9a3412",
  warningBg: "#fff7ed",
  warningBorder: "#fed7aa",
  info: "#1d4ed8",
  infoBg: "#eff6ff",
  infoBorder: "#bfdbfe",
  phi: "#3730a3",
  phiBg: "#eef2ff",
  phiBorder: "#c7d2fe",
  error: "#b91c1c",
<<<<<<< HEAD
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
=======
  errorBg: "#fff1f2",
  errorBorder: "#fecdd3",
} as const;

export const spacing = {
  '2xs': 4,
  xs: 8,
  sm: 12,
  md: 16,
  lg: 20,
  xl: 24,
  '2xl': 32,
  '3xl': 40,
  '4xl': 56,
>>>>>>> af5074b (Sync active ZorbaHealth changes)
} as const;

export const radii = {
  sm: 12,
  md: 16,
  lg: 20,
  xl: 24,
  pill: 999,
} as const;

export const typeScale = {
  overline: 12,
  caption: 13,
  bodySmall: 15,
  body: 17,
  lg: 18,
  h3: 19,
  h2: 22,
  xl: 24,
  hero: 28,
  display: 34,
} as const;

export const fontFamilies = {
  heading: "System",
  body: "System",
  mono: "Menlo",
} as const;

export const shadows = {
  low: {
    shadowColor: colors.text,
    shadowOpacity: 0.14,
    shadowRadius: 24,
    shadowOffset: { width: 0, height: 8 },
    elevation: 3,
  },
  medium: {
    shadowColor: colors.text,
    shadowOpacity: 0.18,
    shadowRadius: 40,
    shadowOffset: { width: 0, height: 18 },
    elevation: 5,
  },
  high: {
    shadowColor: colors.text,
    shadowOpacity: 0.22,
    shadowRadius: 70,
    shadowOffset: { width: 0, height: 28 },
    elevation: 8,
  },
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

export const motion = {
  patient: {
    easing: [0.4, 0, 0.2, 1],
    durations: {
      micro: 120,
      normal: 360,
      macro: 520,
    },
    entrance: {
      pattern: "Card Entrance (Premium)",
      translateY: 20,
    },
  },
  hospital: {
    easing: [0.2, 0, 0, 1],
    durations: {
      micro: 100,
      normal: 240,
      macro: 360,
    },
    entrance: {
      pattern: "Corporate triage update",
      overshoot: 0.03,
    },
  },
} as const;

export const touchTargets = {
  minimum: 44,
  comfortable: 48,
} as const;

export const semanticColors = {
  success: {
    fg: colors.success,
    bg: colors.successBg,
    border: colors.successBorder,
  },
  warning: {
    fg: colors.warning,
    bg: colors.warningBg,
    border: colors.warningBorder,
  },
  info: {
    fg: colors.info,
    bg: colors.infoBg,
    border: colors.infoBorder,
  },
  error: {
    fg: colors.error,
    bg: colors.errorBg,
    border: colors.errorBorder,
  },
  phi: {
    fg: colors.phi,
    bg: colors.phiBg,
    border: colors.phiBorder,
  },
} as const;
