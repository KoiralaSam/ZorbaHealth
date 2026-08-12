# From product to prototype to peak UX

Use Codex to reconstruct the current ZorbaHealth experience as a Figma prototype, redesign it into a best-in-class modern medical product, then land the redesign in the real web and mobile codebases — in stages, with a goal that keeps the finish line stable.

Skills and MCP used throughout: **product design**, **figma** (Figma MCP), **build web apps**, **build react native apps**, **expo** (Expo MCP), **github** (GitHub MCP), **design-dna**, **ui-ux-pro-max**, **motion-design**, and the **GSAP** skill family (`gsap-core`, `gsap-timeline`, `gsap-scrolltrigger`, `gsap-react`, `gsap-plugins`, `gsap-utils`, `gsap-performance`).

**What the newly added skills contribute, so prompts below invoke the right one for the right job:**

- **design-dna** — structures a design identity into three checkable dimensions: `design_system` (measurable tokens), `design_style` (qualitative feel), `visual_effects` (motion/Canvas/WebGL). Used to document the current app's DNA (Phase 2: analyze) and to define the Peak UX DNA as one portable JSON artifact (Phase 1 schema + Phase 3 generate) that both Figma and the eventual code stay accountable to.
- **ui-ux-pro-max** — a searchable rule/pattern database (`scripts/search.py --design-system`, `--domain <ux|color|typography|gsap|react|web>`, `--stack <nextjs|react-native>`) plus CRITICAL→LOW priority checklists for accessibility, touch targets, forms, navigation, and motion. Used to generate the Peak UX design system with reasoning and to run pre-delivery quality gates.
- **motion-design** — timing/easing tables, the four Motion Personality archetypes (Playful/Premium/Corporate/Energetic), the three-motion-layer rule (primary/secondary/ambient), and Disney-principle choreography. Used to pick ZorbaHealth's motion identity once and apply it consistently everywhere.
- **GSAP skills** — the implementation layer for web motion (DOM/SVG only): `gsap-core` for tweens/easing, `gsap-timeline` for sequencing, `gsap-scrolltrigger` for scroll-linked reveals, `gsap-react` for `useGSAP`/cleanup, `gsap-plugins`/`gsap-utils` for Flip/Draggable/helpers, `gsap-performance` for 60fps budgets. GSAP does not run natively in Expo/React Native — mobile motion follows the same motion-design identity but is implemented with React Native's own animation APIs, not GSAP.

---

## 01 — Initial prompt

Start by building a faithful "design twin" of what already ships. Codex reads the running code, not screenshots from memory, so the prototype matches reality.

### ZorbaHealth Design Twin — initial prompt

```text
Use the product design and figma skills with the Figma MCP to build a complete,
navigable Figma prototype of the CURRENT ZorbaHealth product — both the Next.js
web app and the Expo React Native app — exactly as they exist in this repository
today. This is an as-is reconstruction, not a redesign. Do not "improve" anything
in this pass.

SOURCE OF TRUTH

Read the actual code before drawing a single frame:

- Web pages: `web/src/app/**/page.tsx`, shared chrome in `web/src/components/`
  (layout, ui, magicui, status-banner, empty-state, theme-toggle).
- Mobile: `mobile/App.tsx` (the entire app lives here), primitives in
  `mobile/src/components/primitives.tsx`, feature components in
  `mobile/src/components/`, and design tokens in `mobile/src/theme/tokens.ts`.
- Also run both apps locally (web dev server; Expo via the Expo MCP) and compare
  each Figma frame against the live screen before calling it done.

FIGMA FILE STRUCTURE

Create one Figma file named "ZorbaHealth — Current State" with these pages:

1. "Foundations" — the existing token set as Figma variables and styles:
   - Colors: primary indigo #4f46e5, deep navy #111c3a, accent orange #fb923c,
     canvas #e8eefc, background #f5f7ff, surface #ffffff, muted surface #eef2ff,
     text #0b1120, subtle #475569, muted #64748b, borders #dce3f4/#dbeafe,
     success #047857 on #ecfdf5, error #b91c1c on #fff8f8, plus the dark
     slate-950 landing palette with indigo/orange radial gradients.
   - Radii 12/16/20/24/pill, spacing 4–32, type scale 11/13/15/18/24/28.
   - Component library: primary button, shimmer CTA, icon button, segmented
     control, tab bar, info card, section card, field, feedback banner,
     empty state, loading card, status banner — matching the shipped primitives.

2. "Web — Marketing & Auth" — desktop 1440 frames:
   - `/` landing: dark hero with aurora headline, dot pattern, shimmer
     "Start patient onboarding" CTA, "Register Hospital" outline CTA,
     stat trio, feature sections, header linking Patient/Hospital portals.
   - `/login/patient`, `/login/hospital`, `/login/hospital_staff`.
   - `/register/patient` → `/register/patient/verify` → `/register/patient/otp`,
     `/verify-email` (success, error, and expired states).
   - `/register/hospital`.

3. "Web — Patient Portal" — `/patient`, "Consent-aware care workspace",
   sidebar with all nine tabs, one frame per tab:
   Dashboard, Consents, Ask Records, Call History, Meetings, Appointments,
   Welfare Checks, Audit Trail, GPS. Include the appointment booking panel
   (hospital picker → calendar → auto-selected earliest slot) and the meeting
   scheduling form exactly as implemented.

4. "Web — Hospital Console" — `/hospital/records/summary` ("Clinical Record
   Manager"), one frame per tab: Home (patient search + incoming bridges),
   Summary (Full / Medications / Allergies / Diagnoses modes), Meetings,
   Appointments (staff availability board), Staff, Consent (QR issuance),
   Incidents, Audit Search.

5. "Web — Meeting" — `/meeting/join`: pre-join, in-call LiveKit room,
   "Visit ended", and "Missing join details" states.

6. "Mobile — Patient" — 390×844 frames for the Expo patient experience:
   login/register/verify/OTP, then the ten-tab portal: Home, Consent, Scan
   (camera QR consent scanner), Ask (records Q&A), Calls, Meetings, Appts
   (hospital consent picker → green-date calendar → earliest open slot),
   Welfare, Audit, GPS (live location websocket status).

7. "Mobile — Hospital" — the staff-side tabs: Home, Summarize, Meetings,
   Appts (WeeklyScheduleBoard), Staff, Consent, Alerts Inbox, Audit Search;
   plus the MeetingVideoRoom in-call screen.

PROTOTYPE WIRING

- Wire every navigation affordance: web sidebar tabs, mobile tab bar, auth
  flows (login → portal, register → verify → OTP → portal), consent QR scan →
  granted state, appointment booking (pick hospital → pick date → confirm →
  booked state), meeting card → join → in-call → ended.
- Reproduce loading, empty ("No linked hospitals yet. Grant hospital consent
  first."), error, and success states for every data surface — these exist in
  code as LoadingCard, EmptyText, Feedback, and status-banner; do not invent
  copy, lift it from the source.

QUALITY BAR

- Every frame uses auto layout and the shared component library — no detached
  one-off styles.
- Text, spacing, colors, and iconography (Ionicons on mobile, Lucide on web)
  must match the running app. When in doubt, screenshot the live screen via
  the Expo MCP / browser and correct the frame.
- Name frames and layers by route and tab (e.g. "web/patient/appointments",
  "mobile/patient/scan") so later iteration prompts can target them precisely.
- Finish by posting a summary comment in the Figma file listing every screen
  covered and any state you could not reach in the live app.

CURRENT DESIGN DNA (documentation only — do not improve anything here)

Use the design-dna skill's Phase 2 (analyze) against the running apps and this
Figma file to produce `docs/design/current-state-dna.json`: populate the
`design_system` dimension from the real tokens above, the `design_style`
dimension from an honest read of the current look (e.g. "utilitarian tab-bar
SaaS," dense, low ornamentation), and the `visual_effects` dimension from what
actually exists in code today — scan for GSAP/`motion`/Reanimated usage,
scroll effects, and transitions; set `enabled: false` for anything absent
rather than implying motion work that hasn't been done. This file is the
"before" baseline prompt 04 will diff against — do not editorialize or
propose fixes in it.
```

---

## 02 — Build an ambitious redesign

Start with the experience you want, then let Codex work through the design system, accessibility, and interaction details needed to make it real.

- **Describe the patient's day, not the screen.** A medication-safety question at 2 a.m., a welfare check for an aging parent, a nurse triaging incidents between rounds — name the moments the design must serve.
- **Anchor on clinical trust.** Medical software earns calm through hierarchy, generous type, and predictable navigation — not through decoration.
- **Let ui-ux-pro-max propose, don't guess.** Run `--design-system` with product/industry/tone keywords (e.g. "healthcare consumer portal calm trustworthy") before hand-picking colors or fonts; it returns a reasoned system plus anti-patterns, and the `--variance`/`--motion`/`--density` dials tune boldness, animation complexity, and spacing without changing the brief.
- **Choose one motion personality and defend it.** Use the motion-design skill's decision tree (functional vs. brand vs. narrative vs. choreographed) to pick a single archetype — likely Premium for patients, Corporate for the hospital console — rather than letting each screen invent its own timing and easing.
- **Ask for the hard version.** WCAG 2.2 AA, dynamic type, one-handed reach, interruptible video visits, and panic-proof emergency affordances are design problems Codex can work through and test.
- **Try it, then adjust it.** Open the prototype, walk each flow as a patient and as a nurse, and steer the next change with specific evidence.

---

## 03 — Build it in stages

1. **Reconstruct the current state.** The design twin (prompt 01) makes the existing UX visible; a design-dna analysis pass gives every later change a structured baseline to diff against, not just screenshots.
2. **Redesign in Figma.** Rework the system and flows for peak medical UX (prompt 04) using ui-ux-pro-max for the reasoned design system and motion-design for the motion identity, captured as a single Peak UX design-dna JSON, while the code keeps shipping untouched.
3. **Land it in code.** Implement the approved direction in `web/` and `mobile/` (prompt 05) — GSAP skills realize the web motion identity; mobile realizes the same motion-design identity through React Native's own animation primitives — one reviewable PR per surface.
4. **Polish against a goal.** Hold the acceptance criteria fixed (prompt 06), run the ui-ux-pro-max pre-delivery checklist and GSAP performance checks, and iterate with real devices and browsers until it feels inevitable.

---

## 04 — Iteration prompt

Redesign the prototype for peak medical UX without discarding the product's real information architecture, flows, or constraints.

### ZorbaHealth Peak UX — iteration prompt

```text
Starting from the "ZorbaHealth — Current State" Figma file, use the product
design and figma skills to create a sibling file "ZorbaHealth — Peak UX" that
redesigns both apps into a benchmark modern medical product. Preserve the real
feature set, routes, roles (patient, hospital admin, hospital staff), and data
states — redesign how they are experienced, not what exists.

SET A GOAL BEFORE YOU START

Before drawing anything, set a goal for this work and keep iterating until it
is met. The goal: "Every core journey in the ZorbaHealth prototype — patient
books an appointment, grants consent via QR, joins a video visit, asks a
records question; staff triages an incident and reviews a summary — completes
in measurably fewer decisions than the current-state file, with every frame
passing WCAG 2.2 AA and every flow walkable end-to-end in Figma prototype
mode." Write the goal's acceptance criteria into the Figma file on a "Goal"
page, count the decision steps per journey in the current-state file first so
the improvement is measurable, and check each criterion off with a linked
walkthrough before declaring the redesign done. If a criterion cannot be met,
record why on the Goal page rather than silently dropping it.

DESIGN INTELLIGENCE INPUTS (run before drawing frames)

1. Read `docs/design/current-state-dna.json` (from prompt 01) so the redesign
   is a deliberate delta, not a blind rebuild.
2. Use the ui-ux-pro-max skill to generate the design system with reasoning:
   `python3 skills/ui-ux-pro-max/scripts/search.py "consumer healthcare
   patient portal calm trustworthy" --design-system --variance 4 --motion 5
   --density 5 -p "ZorbaHealth Patient"` for the patient surface, and a second
   run with `"clinical staff console dense fast triage"` and `--density 8` for
   the hospital console — two rhythms, one system. Follow with
   `--domain color "healthcare trust"`, `--domain typography "clinical
   calm legible"`, and `--domain ux "accessibility forms navigation"` to fill
   in details the design-system pass didn't cover.
3. Use the motion-design skill to select ONE Motion Personality archetype per
   surface (Premium for patient, Corporate for hospital — see the archetype
   table) and write down the resulting signature easing curve, 3-duration
   palette, and entrance pattern; every animation decision below must trace
   back to this choice.
4. Use the design-dna skill (Phase 1 schema, then Phase 3 generate) to
   consolidate steps 1–3 into `docs/design/peak-ux-dna.json` — one structured
   artifact covering `design_system` (the ui-ux-pro-max tokens),
   `design_style` (the qualitative direction below), and `visual_effects`
   (the motion-design personality + choreography rules, marked for GSAP on
   web / native animation APIs on mobile). This JSON is the single source of
   truth prompt 05 implements against — Figma and code must not drift from it
   independently.

DESIGN PRINCIPLES

- Calm clinical confidence: a light, airy patient surface and a denser,
  faster hospital console — same system, two rhythms.
- One glance, one truth: every screen answers "what needs my attention now?"
  above the fold before offering anything else.
- Consent is the spine: consent state must be visible and reversible from
  anywhere PHI appears, never buried in a settings tab.
- Nothing scary by accident: destructive and medical-risk actions (revoking
  consent, cancelling welfare checks, escalations) get deliberate friction;
  everything else gets less.

DESIGN SYSTEM EVOLUTION

- Evolve the token set rather than replacing it: keep indigo as the trust
  primary, retire the orange accent to a warm highlight role, add a clinical
  semantic layer (info / success / caution / critical / PHI-sensitive) with
  AA-contrast pairings in both light and dark themes.
- Type: move to a two-family system (a humanist sans for UI, a slightly
  warmer face for patient-facing headlines), minimum 16px body on web and
  17pt on mobile, full dynamic-type support.
- Build the components as Figma variants with all interactive states
  (default, hover, focus-visible, pressed, disabled, loading, error) and
  document usage rules on the Foundations page.

PATIENT EXPERIENCE (web + mobile)

- Replace the flat nine/ten-tab list with a task-first structure:
  Home, Care (meetings + appointments + calls unified as a timeline),
  Records (ask + summaries), Privacy (consents + scan + audit), and a
  persistent emergency/welfare affordance that is reachable in one gesture.
- Home becomes a "next best action" surface: upcoming visit with join
  countdown, pending consent requests, unread care events — not a grid of
  equal cards.
- Appointment booking collapses to three decisions (who, when, confirm) with
  the earliest-slot default made visible and overridable, and timezone shown
  in plain language.
- The QR consent scan flow gets a first-run explainer, a clear "what you are
  granting" summary before confirmation, and an instant undo after.
- Video visits: pre-join device check, captions toggle, and a degraded-mode
  path (audio-only) designed for poor connectivity.
- Annotate each transition (tab switch, card entrance, consent-granted
  success state, appointment confirmation) with the motion-design pattern it
  follows — e.g. "Card Entrance (Premium)" or "Success State" from the
  skill's Common Patterns — so prompt 05 implements a named pattern, not a
  guess. Apply the three-motion-layer rule (primary + secondary + ambient) to
  the Home "next best action" card and the appointment-confirmed state.

HOSPITAL EXPERIENCE (web + mobile)

- The console leads with a triage strip: incidents, incoming bridge calls,
  and today's schedule in one row, always visible.
- Record summaries present the AI output with provenance (which records,
  which consent), a medications/allergies/diagnoses filter as segmented
  control, and a visible "AI-generated — verify before acting" posture.
- Audit search becomes investigable: filter chips, timeline view, and
  export affordance.
- The triage strip's live-update motion (new incident/bridge call arriving)
  follows the Corporate archetype: 200–400ms, `cubic-bezier(0.2,0,0,1)`, 0–3%
  overshoot — urgent enough to notice, controlled enough not to feel alarming
  on every refresh.

ACCESSIBILITY AND TRUST (non-negotiable)

- WCAG 2.2 AA contrast on every text/background pairing in both themes —
  cross-check with ui-ux-pro-max's `color-accessible-pairs` and
  `color-contrast` rules (Quick Reference §1) rather than eyeballing it.
- Visible focus order annotated on every frame; touch targets ≥ 44×44pt per
  `touch-target-size` (Quick Reference §2).
- Reduced-motion variants for any animated surface, per motion-design's
  quality rule against opacity-only state changes and ui-ux-pro-max's
  `reduced-motion`/`excessive-motion` rules — never more than 1–2 elements
  animating per view.
- Plain-language microcopy at ~8th-grade reading level for all patient-facing
  text; keep clinical precision on the hospital side.
- Annotate every PHI-bearing region so engineering knows what must never
  appear in logs, notifications, or previews.
- Run `--domain ux "animation accessibility z-index loading"` as a design
  validation pass over the finished file before moving to delivery.

DELIVERY

- Wire the full prototype with the same flow coverage as the current-state
  file, plus the new flows (emergency affordance, consent explainer, pre-join
  check).
- Produce a "Redesign rationale" page: before/after thumbnails per screen,
  the UX problem each change solves, and anything intentionally deferred.
- Keep frame names route-aligned with the current-state file so the
  implementation diff is mechanical.
- Commit `docs/design/peak-ux-dna.json` alongside the Figma file — it is the
  handoff artifact prompt 05 reads, not just this file's internal record.
```

---

## 05 — Iteration prompt

Land the approved design in the real codebases without redesigning the backend contracts, routes, or data flows that already work.

### ZorbaHealth Implementation — iteration prompt

```text
Implement the "ZorbaHealth — Peak UX" Figma direction in this repository using
the build web apps and build react native apps skills. Use the Figma MCP to
read exact tokens, spacing, and component specs from the file, and treat
`docs/design/peak-ux-dna.json` (written in prompt 04) as the source of truth
for tokens, motion personality, and effects — do not eyeball either.

SET A GOAL BEFORE YOU START

Set a goal for this implementation and work against it across all stages: "All
five staged PRs are open and review-ready; every changed screen matches its
Figma frame; zero hardcoded colors, radii, or font sizes remain in touched
components; and every pre-existing flow (login → portal, QR consent grant,
appointment booking, meeting join) still completes against the local stack."
Turn that goal into a checklist in a tracking GitHub issue via the GitHub MCP
before the first commit, link each staged PR to it, and update it as evidence
accumulates. Do not treat any stage as finished because it compiles — a stage
is finished when its slice of the goal checklist has screenshots or a recorded
walkthrough attached. Keep iterating until every box is checked or explicitly
descoped with a reason in the issue.

GROUND RULES

- Frontend-only: do not change API gateway routes, gRPC contracts, proto
  files, or backend services. All existing endpoints keep working unchanged.
- Web work stays in `web/` (Next.js App Router + Tailwind); mobile work stays
  in `mobile/` (Expo). Respect the repo architecture rules.
- Evolve `mobile/src/theme/tokens.ts` and the web Tailwind theme into the new
  token set first, in its own commit, so every subsequent change is expressed
  in tokens — no hardcoded hex values in components.
- Decompose as you go: the mobile app currently lives in a ~5000-line
  `App.tsx`. Extract each redesigned screen into `mobile/src/screens/` with
  the existing primitives pattern, but do not rewrite data fetching or auth
  logic — move it, don't change it.
- Before writing implementation code for each surface, run
  `python3 skills/ui-ux-pro-max/scripts/search.py "<surface keywords>"
  --stack nextjs` (web) or `--stack react-native` (mobile) for
  implementation-specific do/don't guidance, and cross-check icons against
  the `icons` domain instead of introducing a new icon set.

MOTION IMPLEMENTATION (GSAP on web, native APIs on mobile)

- Web (`web/`): implement the `visual_effects` motion personality from
  `peak-ux-dna.json` with the GSAP skills — `gsap-core` for the signature
  easing/duration tokens as reusable constants, `gsap-timeline` for
  multi-step sequences (booking confirmation, consent-granted), and
  `gsap-scrolltrigger` only where the redesign calls for scroll-linked reveal
  (e.g. the audit timeline). Use `gsap-react`'s `useGSAP()` hook for all
  component-level animation so cleanup on unmount is automatic — do not use
  raw `useEffect` for GSAP. Apply `gsap-performance` guidance: animate only
  `transform`/`opacity`, batch triggers, and keep the per-frame budget inside
  60fps. The existing `motion` (Framer Motion) dependency in `web/package.json`
  may stay for simple declarative cases, but any sequenced, scroll-linked, or
  interruptible animation goes through GSAP so there is one motion engine of
  record, not two competing ones.
- Mobile (`mobile/`): GSAP does not run natively in Expo/React Native. Realize
  the SAME motion personality (same durations, same easing feel, same
  three-layer choreography) using React Native's `Animated` API or
  `react-native-reanimated` if you add it — document which one and why in the
  PR description. Do not silently drop the motion identity because the tool
  differs; translate it deliberately.
- Every new animation must cite which motion-design pattern it implements
  (from the Figma annotations in prompt 04) in a code comment or commit
  message — no unattributed animation code.

STAGING AND GITHUB

Use the github skill / GitHub MCP to deliver reviewable stages, one branch and
PR each, in this order:

1. `design/tokens` — shared token evolution (web theme + mobile tokens),
   zero visual regressions intended beyond color/type refinement.
2. `design/web-patient` — patient portal restructure (task-first navigation,
   new Home, booking flow, consent surfaces).
3. `design/web-hospital` — hospital console (triage strip, summary
   provenance, audit search).
4. `design/mobile-patient` — Expo patient experience including the scan
   explainer and emergency affordance.
5. `design/mobile-hospital` — Expo staff experience and video room polish.

Each PR description must embed before/after screenshots and link the exact
Figma frames it implements.

VERIFICATION PER STAGE

- Web: run the dev server and inspect each changed route in the browser at
  390, 768, 1280, and 1440 widths, light and dark themes. Production build
  must pass with a clean console.
- Mobile: use the Expo MCP to launch the app, walk every changed tab on an
  iOS and an Android viewport, and screenshot each screen. The Expo bundle
  must build without warnings introduced by this work.
- All existing flows must still complete against the real local stack:
  patient login → portal, consent grant via QR, appointment booking,
  meeting join. If a flow cannot be exercised locally, say so in the PR
  rather than claiming it verified.
- Run the ui-ux-pro-max Pre-Delivery Checklist (Visual Quality, Interaction,
  Light/Dark Mode, Layout, Accessibility) against every changed screen before
  opening the PR; list any unchecked item explicitly rather than omitting it.
- On web, confirm no GSAP tween leaks past unmount (check for repeated
  listeners after navigating away and back) and that Chrome's Performance
  panel shows no dropped frames during the busiest sequenced animation
  (booking confirmation or the triage strip update).
```

---

## 06 — Keep iterating with a goal

A goal keeps the outcome and acceptance criteria stable while Codex continues working. Use new prompts to steer the current work without redefining the finish line.

### ZorbaHealth Peak UX — polish goal

```text
GOAL

Set this as your standing goal and keep working toward it across as many
passes as needed — do not wait for further prompts to continue: take the
redesigned ZorbaHealth web and mobile apps to a finished, demo-quality level
that matches the "ZorbaHealth — Peak UX" Figma file. Preserve the implemented
navigation structure, token system, feature set, and backend contracts.
Continue iterating through real browser and Expo device inspection until
every screen feels calm, trustworthy, and unmistakably modern — not merely
restyled.

Treat every section below as the goal's acceptance criteria. Maintain a
living checklist of them in the tracking GitHub issue from the implementation
stage; each pass, pick the highest-impact unmet criterion, fix it, attach
evidence, and re-verify neighboring criteria you may have disturbed. The goal
is met only when every criterion has evidence attached.

FIDELITY

- Every implemented screen matches its Figma frame in hierarchy, spacing
  rhythm, and semantic color usage. Where code and Figma disagree, fix the
  code unless the Figma frame is impossible; then update the frame and note
  it in the rationale page.
- No hardcoded colors, radii, or font sizes remain in changed components —
  tokens only, verified by search.
- Light and dark themes are both complete on web; mobile respects the system
  appearance setting.
- `docs/design/peak-ux-dna.json` still accurately describes what's shipped —
  if implementation deviated from any `design_system`, `design_style`, or
  `visual_effects` field, update the JSON (and note why in the rationale
  page) rather than letting it go stale.

MOTION AND DESIGN SYSTEM INTEGRITY

- Every animation in the product traces to the single motion personality
  chosen in prompt 04 (Premium for patient, Corporate for hospital) — no
  screen invents its own duration or easing. Spot-check by grepping for
  animation durations/easings on web and mobile and confirming they map to
  the documented palette.
- Motion-design's CRITICAL quality rules hold everywhere: no linear easing
  for spatial movement, no opacity-only transitions for important state
  changes, no motion exceeding 1/3 of the screen without an intermediate
  keyframe, and every non-trivial animation has primary + secondary + ambient
  layers (check the Home next-best-action card and the appointment-confirmed
  state specifically — these were called out in prompt 04).
- On web, every GSAP animation is registered through `useGSAP()` (or
  equivalently cleaned up) — audit for orphaned tweens/ScrollTriggers on
  repeated navigation. `gsap-performance` budgets hold at 60fps on the
  heaviest sequenced screen.
- Re-run the ui-ux-pro-max Pre-Delivery Checklist end-to-end (not just
  per-PR) now that all five stages are merged, since cross-stage regressions
  (e.g. a shared component drifting between the patient and hospital PRs)
  only show up once everything is integrated.

EXPERIENCE QUALITY

- Patient Home surfaces the next best action correctly for these states:
  meeting joinable now, meeting later today, pending consent request, no
  activity. Verify each by seeding the local stack.
- The emergency/welfare affordance is reachable in one gesture from every
  patient screen on mobile and never obscured by keyboards or modals.
- Consent revocation, welfare-check cancellation, and meeting cancellation
  all require deliberate confirmation and communicate their consequence in
  plain language; no other action shows a confirm dialog.
- The hospital triage strip updates live for incoming bridge calls and
  incidents without a manual refresh.
- Appointment booking completes in three decisions; the earliest-slot
  default is visible and overridable; timezone reads in plain language.
- Video visit pre-join shows camera/mic state before entry; the ended and
  missing-details states match the redesigned frames.

ACCESSIBILITY

- Keyboard-only completion of login, booking, consent grant/revoke, and
  audit search on web, with visible focus at every step.
- Screen-reader labels on every interactive element in changed mobile
  screens; test with the Expo MCP accessibility inspection.
- All text/background pairings pass AA in both themes; touch targets ≥ 44pt;
  reduced-motion preference disables all non-essential animation.
- Patient-facing copy verified at ~8th-grade reading level.

SAFETY

- No PHI in console logs, toasts, notification previews, or error messages
  introduced by this work. Audit every changed data surface.
- AI-generated summaries always render with the provenance and verify-first
  treatment; there is no path that presents them as plain facts.

VALIDATION LOOP

Do not stop after a successful build.

- Web: walk every route at 390×844, 768, 1280×720, and 1440×900, both
  themes, forward and back navigation, with a clean console. Production
  build and lint must pass.
- Mobile: via the Expo MCP, walk every tab on iOS and Android viewports,
  rotate where supported, background/foreground during a video visit, and
  screenshot each final screen.
- Exercise the real flows end-to-end against the local stack: register →
  verify → OTP → portal; QR consent scan → grant → revoke; book →
  reschedule → cancel appointment; schedule → join → end meeting; trigger a
  welfare check and observe the patient and hospital views.
- File one GitHub issue per remaining defect with a screenshot and the
  Figma frame reference; fix, re-verify, and close each. The goal is done
  when the issue list is empty and every checklist item above has evidence.
```
