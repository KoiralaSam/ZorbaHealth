# Peak UX Design Intelligence Evidence

Generated: 2026-08-12

These are the local ui-ux-pro-max searches named by the Peak UX brief. The final Peak UX DNA keeps ZorbaHealth indigo as the trust primary rather than copying every raw recommendation verbatim.

## patient_design_system

```bash
python3 .agents/skills/ui-ux-pro-max/scripts/search.py consumer healthcare patient portal calm trustworthy --design-system --variance 4 --motion 5 --density 5 -p ZorbaHealth Patient
```

Exit status: 0

```text
╔═════════════════════════════════════════════════════════════════════════════════════════╗
║  TARGET: ZorbaHealth Patient - RECOMMENDED DESIGN SYSTEM                                ║
╚═════════════════════════════════════════════════════════════════════════════════════════╝
┌─────────────────────────────────────────────────────────────────────────────────────────┐
├─── DESIGN DIALS ─────────────────────────────────────────────────────────────────────────┤
│  Variance: 4/10 — Balanced / Modern                                                     │
│  Motion:   5/10 — Standard                                                              │
│  Density:  5/10 — Standard                                                              │
├─── PATTERN ──────────────────────────────────────────────────────────────────────────────┤
│  Name: Enterprise Gateway                                                               │
│     Conversion: Path selection (I am a...). Mega menu navigation. Trust signals prominent.│
│     CTA: Contact Sales (Primary) + Login (Secondary)                                    │
│     Sections:                                                                           │
│       1. 1. Hero (Video/Mission), 2. Solutions by Industry, 3. Solutions by Role, 4. Client Logos, 5. Contact Sales│
├─── STYLE ────────────────────────────────────────────────────────────────────────────────┤
│  Name: Tactile Digital / Deformable UI                                                  │
│     Mode Support: Light ✓ Full  Dark ✓ Full                                             │
│     Keywords: Jelly buttons, chrome, clay, squishy, deformable, bouncy, physical,       │
│     tactile feedback, press response                                                    │
│     Best For: Modern mobile apps, playful brands, entertainment, gaming UI, consumer    │
│     products, interactive demos                                                         │
│     Performance: ⚠ Good | Accessibility: ⚠ Motion sensitive                             │
├─── COLORS ───────────────────────────────────────────────────────────────────────────────┤
│     Primary:       #0891B2    (--color-primary)                                         │
│     On Primary:    #FFFFFF    (--color-on-primary)                                      │
│     Secondary:     #22D3EE    (--color-secondary)                                       │
│     Accent/CTA:    #059669    (--color-accent)                                          │
│     Background:    #ECFEFF    (--color-background)                                      │
│     Foreground:    #164E63    (--color-foreground)                                      │
│     Muted:         #E8F1F6    (--color-muted)                                           │
│     Border:        #A5F3FC    (--color-border)                                          │
│     Destructive:   #DC2626    (--color-destructive)                                     │
│     Ring:          #0891B2    (--color-ring)                                            │
│     Notes: Calm cyan + health green                                                     │
├─── TYPOGRAPHY ───────────────────────────────────────────────────────────────────────────┤
│  Figtree / Noto Sans                                                                    │
│     Mood: medical, clean, accessible, professional, healthcare, trustworthy             │
│     Best For: Healthcare, medical clinics, pharma, health apps, accessibility           │
│     Google Fonts: https://fonts.googleapis.com/css2?family=Figtree:wght@300;400;500;600;700&family=Noto+Sans:wght@300;400;500;700&display=swap│
│     CSS Import: @import url('https://fonts.googleapis.com/css2?family=Figtree:wght@300...│
├─── KEY EFFECTS ──────────────────────────────────────────────────────────────────────────┤
│     Press deformation (scale + squish), bounce-back (cubic-bezier), material response,  │
│     haptic-like feedback, spring physics                                                │
├─── MOTION ───────────────────────────────────────────────────────────────────────────────┤
│  Stagger List (Standard)                                                                │
│     Trigger: load or scroll | Duration: 300-450ms | Easing: back.out(1.4)               │
│     GSAP: gsap.from('.grid-item', { opacity: 0, scale: 0.92, y: 16, duration: 0.4,      │
│     stagger: { each: 0.06, from: 'start', grid: 'auto' }, ease: 'back.out(1.4)' });     │
│     Framework: grid: 'auto' lets GSAP infer rows/columns from a CSS grid layout for a   │
│     natural wave stagger                                                                │
├─── AVOID ────────────────────────────────────────────────────────────────────────────────┤
│     Bright neon colors + Motion-heavy animations + AI purple/pink gradients             │
├─── PRE-DELIVERY CHECKLIST ───────────────────────────────────────────────────────────────┤
│     [ ] No emojis as icons (use SVG: Heroicons/Lucide)                                  │
│     [ ] cursor-pointer on all clickable elements                                        │
│     [ ] Hover states with smooth transitions (150-300ms)                                │
│     [ ] Light mode: text contrast 4.5:1 minimum                                         │
│     [ ] Focus states visible for keyboard nav                                           │
│     [ ] prefers-reduced-motion respected                                                │
│     [ ] Responsive: 375px, 768px, 1024px, 1440px                                        │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

## hospital_design_system

```bash
python3 .agents/skills/ui-ux-pro-max/scripts/search.py clinical staff console dense fast triage --design-system --variance 4 --motion 5 --density 8 -p ZorbaHealth Hospital
```

Exit status: 0

```text
╔═════════════════════════════════════════════════════════════════════════════════════════╗
║  TARGET: ZorbaHealth Hospital - RECOMMENDED DESIGN SYSTEM                               ║
╚═════════════════════════════════════════════════════════════════════════════════════════╝
┌─────────────────────────────────────────────────────────────────────────────────────────┐
├─── DESIGN DIALS ─────────────────────────────────────────────────────────────────────────┤
│  Variance: 4/10 — Balanced / Modern                                                     │
│  Motion:   5/10 — Standard                                                              │
│  Density:  8/10 — Dense / Dashboard                                                     │
├─── PATTERN ──────────────────────────────────────────────────────────────────────────────┤
│  Name: Portfolio Grid                                                                   │
│     Conversion: Visuals first. Filter by category. Fast loading essential.              │
│     CTA: Project Card Hover + Footer Contact                                            │
│     Sections:                                                                           │
│       1. 1. Hero (Name/Role), 2. Project Grid (Masonry), 3. About/Philosophy, 4. Contact│
├─── STYLE ────────────────────────────────────────────────────────────────────────────────┤
│  Name: Soft UI Evolution                                                                │
│     Mode Support: Light ✓ Full  Dark ✓ Full                                             │
│     Keywords: Evolved soft UI, better contrast, modern aesthetics, subtle depth,        │
│     accessibility-focused, improved shadows, hybrid                                     │
│     Best For: Modern enterprise apps, SaaS platforms, health/wellness, modern business  │
│     tools, professional, hybrid                                                         │
│     Performance: ⚡ Excellent | Accessibility: ✓ WCAG AA+                                │
├─── COLORS ───────────────────────────────────────────────────────────────────────────────┤
│     Primary:       #0284C7    (--color-primary)                                         │
│     On Primary:    #FFFFFF    (--color-on-primary)                                      │
│     Secondary:     #0891B2    (--color-secondary)                                       │
│     Accent/CTA:    #16A34A    (--color-accent)                                          │
│     Background:    #F0F9FF    (--color-background)                                      │
│     Foreground:    #0C4A6E    (--color-foreground)                                      │
│     Muted:         #E8F2F8    (--color-muted)                                           │
│     Border:        #BAE6FD    (--color-border)                                          │
│     Destructive:   #DC2626    (--color-destructive)                                     │
│     Ring:          #0284C7    (--color-ring)                                            │
│     Notes: Clinical blue + health green + alert red                                     │
├─── TYPOGRAPHY ───────────────────────────────────────────────────────────────────────────┤
│  Inter / Inter                                                                          │
│     Mood: Friendly + Playful typography                                                 │
├─── KEY EFFECTS ──────────────────────────────────────────────────────────────────────────┤
│     Improved shadows (softer than flat, clearer than neumorphism), modern (200-300ms),  │
│     focus visible, WCAG AA/AAA                                                          │
├─── MOTION ───────────────────────────────────────────────────────────────────────────────┤
│  Stagger List (Standard)                                                                │
│     Trigger: load or scroll | Duration: 300-450ms | Easing: back.out(1.4)               │
│     GSAP: gsap.from('.grid-item', { opacity: 0, scale: 0.92, y: 16, duration: 0.4,      │
│     stagger: { each: 0.06, from: 'start', grid: 'auto' }, ease: 'back.out(1.4)' });     │
│     Framework: grid: 'auto' lets GSAP infer rows/columns from a CSS grid layout for a   │
│     natural wave stagger                                                                │
├─── AVOID ────────────────────────────────────────────────────────────────────────────────┤
│     Generic design + Hidden safety info                                                 │
├─── PRE-DELIVERY CHECKLIST ───────────────────────────────────────────────────────────────┤
│     [ ] No emojis as icons (use SVG: Heroicons/Lucide)                                  │
│     [ ] cursor-pointer on all clickable elements                                        │
│     [ ] Hover states with smooth transitions (150-300ms)                                │
│     [ ] Light mode: text contrast 4.5:1 minimum                                         │
│     [ ] Focus states visible for keyboard nav                                           │
│     [ ] prefers-reduced-motion respected                                                │
│     [ ] Responsive: 375px, 768px, 1024px, 1440px                                        │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

## color_healthcare_trust

```bash
python3 .agents/skills/ui-ux-pro-max/scripts/search.py healthcare trust --domain color
```

Exit status: 0

```text
## UI Pro Max Search Results
**Domain:** color | **Query:** healthcare trust
**Source:** colors.csv | **Found:** 3 results

### Result 1
- **Product Type:** Healthcare App
- **Primary:** #0891B2
- **On Primary:** #FFFFFF
- **Secondary:** #22D3EE
- **On Secondary:** #0F172A
- **Accent:** #059669
- **On Accent:** #FFFFFF
- **Background:** #ECFEFF
- **Foreground:** #164E63
- **Card:** #FFFFFF
- **Card Foreground:** #164E63
- **Muted:** #E8F1F6
- **Muted Foreground:** #64748B
- **Border:** #A5F3FC
- **Destructive:** #DC2626
- **On Destructive:** #FFFFFF
- **Ring:** #0891B2
- **Notes:** Calm cyan + health green

### Result 2
- **Product Type:** Fintech/Crypto
- **Primary:** #F59E0B
- **On Primary:** #0F172A
- **Secondary:** #FBBF24
- **On Secondary:** #0F172A
- **Accent:** #8B5CF6
- **On Accent:** #FFFFFF
- **Background:** #0F172A
- **Foreground:** #F8FAFC
- **Card:** #222735
- **Card Foreground:** #F8FAFC
- **Muted:** #272F42
- **Muted Foreground:** #94A3B8
- **Border:** #334155
- **Destructive:** #EF4444
- **On Destructive:** #FFFFFF
- **Ring:** #F59E0B
- **Notes:** Gold trust + purple tech

### Result 3
- **Product Type:** Legal Services
- **Primary:** #1E3A8A
- **On Primary:** #FFFFFF
- **Secondary:** #1E40AF
- **On Secondary:** #FFFFFF
- **Accent:** #B45309
- **On Accent:** #FFFFFF
- **Background:** #F8FAFC
- **Foreground:** #0F172A
- **Card:** #FFFFFF
- **Card Foreground:** #0F172A
- **Muted:** #E9EEF5
- **Muted Foreground:** #64748B
- **Border:** #CBD5E1
- **Destructive:** #DC2626
- **On Destructive:** #FFFFFF
- **Ring:** #1E3A8A
- **Notes:** Authority navy + trust gold
```

## typography_clinical_calm_legible

```bash
python3 .agents/skills/ui-ux-pro-max/scripts/search.py clinical calm legible --domain typography
```

Exit status: 0

```text
## UI Pro Max Search Results
**Domain:** typography | **Query:** clinical calm legible
**Source:** typography.csv | **Found:** 3 results

### Result 1
- **Font Pairing Name:** Wellness Calm
- **Category:** Serif + Sans
- **Heading Font:** Lora
- **Body Font:** Raleway
- **Mood/Style Keywords:** calm, wellness, health, relaxing, natural, organic
- **Best For:** Health apps, wellness, spa, meditation, yoga, organic brands
- **Google Fonts URL:** https://fonts.googleapis.com/css2?family=Lora:wght@400;500;600;700&family=Raleway:wght@300;400;500;600;700&display=swap
- **CSS Import:** @import url('https://fonts.googleapis.com/css2?family=Lora:wght@400;500;600;700&family=Raleway:wght@300;400;500;600;700&display=swap');
- **Tailwind Config:** fontFamily: { serif: ['Lora', 'serif'], sans: ['Raleway', 'sans-serif'] }
- **Notes:** Lora's organic curves with Raleway's elegant simplicity.

### Result 2
- **Font Pairing Name:** Spatial Clear
- **Category:** Sans + Sans
- **Heading Font:** Inter
- **Body Font:** Inter
- **Mood/Style Keywords:** spatial, legible, glass, system, clean, neutral
- **Best For:** Spatial computing, AR/VR, glassmorphism interfaces
- **Google Fonts URL:** https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600&display=swap
- **CSS Import:** @import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600&display=swap');
- **Tailwind Config:** fontFamily: { sans: ['Inter', 'sans-serif'] }
- **Notes:** Optimized for readability on dynamic backgrounds.

### Result 3
- **Font Pairing Name:** Enterprise SaaS Mobile (Plus Jakarta Sans)
- **Category:** Geometric Sans (Single Family)
- **Heading Font:** Plus Jakarta Sans
- **Body Font:** Plus Jakarta Sans
- **Mood/Style Keywords:** enterprise, saas, b2b, professional, indigo, modern, approachable, legible, ios dynamic type, android scaling
- **Best For:** B2B SaaS apps, productivity tools, government and finance mobile apps, admin dashboards, enterprise onboarding
- **Google Fonts URL:** https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:ital,wght@0,400;0,600;0,700;0,800;1,400
- **CSS Import:** @import url('https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:ital,wght@0,400;0,600;0,700;0,800;1,400&display=swap');
- **Tailwind Config:** fontFamily: { sans: ['Plus Jakarta Sans', 'sans-serif'] }
- **Notes:** Single-family system: Plus Jakarta Sans balances professional authority with mobile approachability. Weight scale: ExtraBold 800 for screen titles/hero (line height 1.1–1.2). Bold 700 for section headers. SemiBold 600 for card titles and buttons. Regular 400 for body text (line height 1.4–1.5). Must...
```

## ux_accessibility_forms_navigation

```bash
python3 .agents/skills/ui-ux-pro-max/scripts/search.py accessibility forms navigation --domain ux
```

Exit status: 0

```text
## UI Pro Max Search Results
**Domain:** ux | **Query:** accessibility forms navigation
**Source:** ux-guidelines.csv | **Found:** 3 results

### Result 1
- **Category:** Accessibility
- **Issue:** Heading Hierarchy
- **Platform:** Web
- **Description:** Screen readers use headings for navigation
- **Do:** Use sequential heading levels h1-h6
- **Don't:** Skip heading levels or misuse for styling
- **Code Example Good:** h1 then h2 then h3
- **Code Example Bad:** h1 then h4
- **Severity:** Medium

### Result 2
- **Category:** Accessibility
- **Issue:** Keyboard Navigation
- **Platform:** Web
- **Description:** All functionality accessible via keyboard
- **Do:** Tab order matches visual order
- **Don't:** Keyboard traps or illogical tab order
- **Code Example Good:** tabIndex for custom order
- **Code Example Bad:** Unreachable elements
- **Severity:** High

### Result 3
- **Category:** Accessibility
- **Issue:** Skip Links
- **Platform:** Web
- **Description:** Allow keyboard users to skip navigation
- **Do:** Provide skip to main content link
- **Don't:** No skip link on nav-heavy pages
- **Code Example Good:** Skip to main content link
- **Code Example Bad:** 100 tabs to reach content
- **Severity:** Medium
```

## ux_animation_accessibility_z_index_loading

```bash
python3 .agents/skills/ui-ux-pro-max/scripts/search.py animation accessibility z-index loading --domain ux
```

Exit status: 0

```text
## UI Pro Max Search Results
**Domain:** ux | **Query:** animation accessibility z-index loading
**Source:** ux-guidelines.csv | **Found:** 3 results

### Result 1
- **Category:** Animation
- **Issue:** Loading States
- **Platform:** All
- **Description:** Show feedback during async operations
- **Do:** Use skeleton screens or spinners
- **Don't:** Leave UI frozen with no feedback
- **Code Example Good:** animate-pulse skeleton
- **Code Example Bad:** Blank screen while loading
- **Severity:** High

### Result 2
- **Category:** Animation
- **Issue:** Continuous Animation
- **Platform:** All
- **Description:** Infinite animations are distracting
- **Do:** Use for loading indicators only
- **Don't:** Use for decorative elements
- **Code Example Good:** animate-spin on loader
- **Code Example Bad:** animate-bounce on icons
- **Severity:** Medium

### Result 3
- **Category:** Layout
- **Issue:** Stacking Context
- **Platform:** Web
- **Description:** New stacking contexts reset z-index
- **Do:** Understand what creates new stacking context
- **Don't:** Expect z-index to work across contexts
- **Code Example Good:** Parent with z-index isolates children
- **Code Example Bad:** z-index: 9999 not working
- **Severity:** Medium
```

