import type { Config } from "tailwindcss";
import defaultTheme from "tailwindcss/defaultTheme";

export default {
    darkMode: ["class"],
    content: [
    "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
    "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
  	extend: {
  		colors: {
  			background: 'hsl(var(--background))',
  			foreground: 'hsl(var(--foreground))',
        card: {
  				DEFAULT: 'hsl(var(--card))',
  				foreground: 'hsl(var(--card-foreground))'
  			},
  			popover: {
  				DEFAULT: 'hsl(var(--popover))',
  				foreground: 'hsl(var(--popover-foreground))'
  			},
  			primary: {
  				DEFAULT: 'hsl(var(--primary))',
  				foreground: 'hsl(var(--primary-foreground))'
  			},
  			secondary: {
  				DEFAULT: 'hsl(var(--secondary))',
  				foreground: 'hsl(var(--secondary-foreground))'
  			},
  			muted: {
  				DEFAULT: 'hsl(var(--muted))',
  				foreground: 'hsl(var(--muted-foreground))'
  			},
  			accent: {
  				DEFAULT: 'hsl(var(--accent))',
  				foreground: 'hsl(var(--accent-foreground))'
  			},
  			destructive: {
  				DEFAULT: 'hsl(var(--destructive))',
  				foreground: 'hsl(var(--destructive-foreground))'
  			},
  			border: 'hsl(var(--border))',
  			input: 'hsl(var(--input))',
  			ring: 'hsl(var(--ring))',
  			chart: {
  				'1': 'hsl(var(--chart-1))',
  				'2': 'hsl(var(--chart-2))',
  				'3': 'hsl(var(--chart-3))',
  				'4': 'hsl(var(--chart-4))',
  				'5': 'hsl(var(--chart-5))'
			},
<<<<<<< HEAD
        surface: { page: "var(--zh-surface-page)", raised: "var(--zh-surface-raised)", subtle: "var(--zh-surface-subtle)" },
        text: { primary: "var(--zh-text-primary)", secondary: "var(--zh-text-secondary)" },
        clinical: { info: "var(--zh-info)", success: "var(--zh-success)", caution: "var(--zh-caution)", critical: "var(--zh-critical)", phi: "var(--zh-phi)" },
        warm: { highlight: "var(--zh-warm-highlight)" },
      },
      borderRadius: {
  			lg: 'var(--radius)',
  			md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
        card: "var(--zh-radius-card)",
        panel: "var(--zh-radius-panel)",
        pill: "var(--zh-radius-pill)",
=======
        canvas: 'hsl(var(--canvas))',
        surface: {
          DEFAULT: 'hsl(var(--surface))',
          muted: 'hsl(var(--surface-muted))',
        },
        subtle: 'hsl(var(--text-subtle))',
        peak: {
          primary: 'hsl(var(--primary))',
          secondary: 'hsl(var(--secondary))',
          warm: 'hsl(var(--accent))',
        },
        success: {
          DEFAULT: 'hsl(var(--success))',
          background: 'hsl(var(--success-background))',
          border: 'hsl(var(--success-border))',
        },
        warning: {
          DEFAULT: 'hsl(var(--warning))',
          background: 'hsl(var(--warning-background))',
          border: 'hsl(var(--warning-border))',
        },
        error: {
          DEFAULT: 'hsl(var(--error))',
          background: 'hsl(var(--error-background))',
          border: 'hsl(var(--error-border))',
        },
        info: {
          DEFAULT: 'hsl(var(--info))',
          background: 'hsl(var(--info-background))',
          border: 'hsl(var(--info-border))',
        },
        phi: {
          DEFAULT: 'hsl(var(--phi))',
          background: 'hsl(var(--phi-background))',
          border: 'hsl(var(--phi-border))',
        },
  		},
  		borderRadius: {
  			lg: 'var(--radius)',
  			md: 'calc(var(--radius) - 2px)',
			sm: 'calc(var(--radius) - 4px)',
        peak: '1.25rem',
        pill: '999px',
  		},
      spacing: {
        'peak-2xs': '0.25rem',
        'peak-xs': '0.5rem',
        'peak-sm': '0.75rem',
        'peak-md': '1rem',
        'peak-lg': '1.25rem',
        'peak-xl': '1.5rem',
        'peak-2xl': '2rem',
        'peak-3xl': '2.5rem',
        'peak-4xl': '3.5rem',
      },
      fontSize: {
        'peak-display': ['2.75rem', { lineHeight: '1.12', fontWeight: '700', letterSpacing: '0' }],
        'peak-h1': ['2rem', { lineHeight: '1.18', fontWeight: '700', letterSpacing: '0' }],
        'peak-h2': ['1.5rem', { lineHeight: '1.25', fontWeight: '700', letterSpacing: '0' }],
        'peak-h3': ['1.25rem', { lineHeight: '1.3', fontWeight: '650', letterSpacing: '0' }],
        'peak-body': ['1rem', { lineHeight: '1.55', fontWeight: '400', letterSpacing: '0' }],
        'peak-body-sm': ['0.875rem', { lineHeight: '1.45', fontWeight: '400', letterSpacing: '0' }],
        'peak-caption': ['0.75rem', { lineHeight: '1.35', fontWeight: '600', letterSpacing: '0' }],
        'peak-overline': ['0.6875rem', { lineHeight: '1.2', fontWeight: '700', letterSpacing: '0.08em' }],
>>>>>>> af5074b (Sync active ZorbaHealth changes)
      },
      fontFamily: {
        sans: ["var(--font-ui)", ...defaultTheme.fontFamily.sans],
        patient: ["var(--font-patient)", ...defaultTheme.fontFamily.serif],
      },
      boxShadow: {
        low: 'var(--shadow-low)',
        medium: 'var(--shadow-medium)',
        high: 'var(--shadow-high)',
        clinical: 'var(--shadow-medium)',
        glow: '0 18px 40px -22px hsl(var(--primary) / 0.65)',
        warm: '0 18px 34px -24px hsl(var(--accent) / 0.75)',
      },
      transitionTimingFunction: {
        'patient': 'var(--motion-patient-ease)',
        'hospital': 'var(--motion-hospital-ease)',
      },
      transitionDuration: {
        'patient-micro': 'var(--motion-patient-micro)',
        'patient-normal': 'var(--motion-patient-normal)',
        'patient-macro': 'var(--motion-patient-macro)',
        'hospital-micro': 'var(--motion-hospital-micro)',
        'hospital-normal': 'var(--motion-hospital-normal)',
        'hospital-macro': 'var(--motion-hospital-macro)',
      },
      keyframes: {
        "fade-in": {
          "0%": { opacity: "0" },
          "100%": { opacity: "1" },
        },
        "fade-in-up": {
          "0%": { opacity: "0", transform: "translateY(16px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
        "slide-up": {
          "0%": { transform: "translateY(10px)" },
          "100%": { transform: "translateY(0)" },
        },
        "pulse-gentle": {
          "0%, 100%": { opacity: "1", transform: "scale(1)" },
          "50%": { opacity: "0.74", transform: "scale(0.98)" },
        },
        shimmer: {
          "0%": { backgroundPosition: "-700px 0" },
          "100%": { backgroundPosition: "700px 0" },
        },
        aurora: {
          from: { backgroundPosition: "50% 50%" },
          to: { backgroundPosition: "350% 50%" },
        },
        "shiny-text": {
          "0%, 90%, 100%": { backgroundPosition: "calc(-100% - var(--shiny-width)) 0" },
          "30%, 60%": { backgroundPosition: "calc(100% + var(--shiny-width)) 0" },
        },
        marquee: {
          from: { transform: "translateX(0)" },
          to: { transform: "translateX(calc(-100% - var(--gap)))" },
        },
        "marquee-vertical": {
          from: { transform: "translateY(0)" },
          to: { transform: "translateY(calc(-100% - var(--gap)))" },
        },
        orbit: {
          "0%": {
            transform:
              "rotate(calc(var(--angle) * 1deg)) translateY(calc(var(--radius) * 1px)) rotate(calc(var(--angle) * -1deg))",
          },
          "100%": {
            transform:
              "rotate(calc((var(--angle) + 360) * 1deg)) translateY(calc(var(--radius) * 1px)) rotate(calc((var(--angle) + -360) * 1deg))",
          },
        },
        "shimmer-slide": {
          to: {
            transform: "translate(calc(100cqw - 100%), 0)",
          },
        },
        "spin-around": {
          "0%": { transform: "translateZ(0) rotate(0)" },
          "15%, 35%": { transform: "translateZ(0) rotate(90deg)" },
          "65%, 85%": { transform: "translateZ(0) rotate(270deg)" },
          "100%": { transform: "translateZ(0) rotate(360deg)" },
        },
      },
      animation: {
        "fade-in": "fade-in 450ms ease-out both",
        "fade-in-up": "fade-in-up 600ms ease-out both",
        "slide-up": "slide-up 300ms ease-out both",
        "pulse-gentle": "pulse-gentle 2.4s ease-in-out infinite",
        shimmer: "shimmer 1.8s linear infinite",
        aurora: "aurora 8s linear infinite",
        "shiny-text": "shiny-text 8s infinite",
        marquee: "marquee var(--duration) linear infinite",
        "marquee-vertical": "marquee-vertical var(--duration) linear infinite",
        orbit: "orbit calc(var(--duration) * 1s) linear infinite",
        "shimmer-slide": "shimmer-slide var(--speed) ease-in-out infinite alternate",
        "spin-around": "spin-around calc(var(--speed) * 2) infinite linear",
      },
  	}
  },
  plugins: [require("tailwindcss-animate")],
} satisfies Config;
