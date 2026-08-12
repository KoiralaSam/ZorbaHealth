# Peak UX Implementation Tracking Issue

GitHub tracking issue: https://github.com/KoiralaSam/ZorbaHealth/issues/42

Created via `gh issue create` on 2026-08-12 because the GitHub MCP issue-creation path returned `403 Resource not accessible by integration`.

## Goal

All five staged PRs are open and review-ready; every changed screen matches its Figma frame; zero hardcoded colors, radii, or font sizes remain in touched components; and every pre-existing flow (login -> portal, QR consent grant, appointment booking, meeting join) still completes against the local stack.

## Source Of Truth

- Peak UX Figma: https://www.figma.com/design/eNq0V4HwW6Yk4ut4GQWg8b
- DNA: `docs/design/peak-ux-dna.json`
- Verification: `docs/design/peak-ux-verification.json`
- WCAG gap audit: `docs/design/peak-ux-wcag-gap-audit.json`

## Staged PR Checklist

- [x] `design/tokens` - shared token evolution in web theme and mobile tokens. Draft PR: https://github.com/KoiralaSam/ZorbaHealth/pull/37
- [x] `design/web-patient` - patient portal restructure. Draft PR: https://github.com/KoiralaSam/ZorbaHealth/pull/38
- [x] `design/web-hospital` - hospital console restructure. Draft PR: https://github.com/KoiralaSam/ZorbaHealth/pull/39
- [x] `design/mobile-patient` - Expo patient experience. Draft PR: https://github.com/KoiralaSam/ZorbaHealth/pull/40
- [x] `design/mobile-hospital` - Expo staff experience and video room polish. Draft PR: https://github.com/KoiralaSam/ZorbaHealth/pull/41

All five staged branches are open as draft PRs and were reported mergeable by `gh pr list` on 2026-08-12. GitHub CI, CodeQL, and GitGuardian checks are green on all five PRs. They stay draft until visual evidence and local flow verification are attached.

## Verification Checklist

- [ ] Web screenshots at 390, 768, 1280, and 1440 widths, light and dark themes.
- [x] Web production build passes; console cleanliness still needs browser-route evidence.
- [ ] Mobile Expo launch and iOS/Android viewport walkthrough screenshots.
- [ ] Existing flows verified: patient login -> portal, QR consent grant, appointment booking, meeting join.
- [ ] ui-ux-pro-max pre-delivery checklist run for every changed screen.
- [ ] Motion maps to Peak UX DNA: Premium for patient, Corporate for hospital.
- [ ] No hardcoded colors, radii, or font sizes remain in touched components.


## Demo-Quality Acceptance Criteria

### Fidelity

- [ ] Every implemented screen matches its Peak UX Figma frame in hierarchy, spacing rhythm, and semantic color usage.
- [ ] No hardcoded colors, radii, or font sizes remain in changed components; tokens only, verified by search.
- [ ] Light and dark themes are complete on web.
- [ ] Mobile respects the system appearance setting.
- [ ] `docs/design/peak-ux-dna.json` accurately describes the shipped implementation, or deviations are documented.

### Motion And Design System Integrity

- [ ] Every animation maps to the documented motion personality: Premium for patient, Corporate for hospital.
- [ ] No linear easing for spatial movement.
- [ ] No opacity-only transitions for important state changes.
- [ ] No motion exceeds 1/3 of the screen without an intermediate keyframe.
- [ ] Non-trivial animations have primary, secondary, and ambient layers where the Peak UX frame calls for them.
- [ ] Web GSAP animation is registered through `useGSAP()` or has equivalent cleanup.
- [ ] Heaviest sequenced screen holds the 60fps performance budget.
- [ ] ui-ux-pro-max pre-delivery checklist is rerun end-to-end after the five staged PRs are integrated.

### Experience Quality

- [ ] Patient Home verifies next-best-action states: meeting joinable now, meeting later today, pending consent request, no activity.
- [ ] Emergency/welfare affordance is reachable in one gesture from every patient mobile screen and is not obscured by keyboards or modals.
- [ ] Consent revocation, welfare-check cancellation, and meeting cancellation require deliberate confirmation with plain-language consequences.
- [ ] Non-risk actions do not show unnecessary confirmation dialogs.
- [ ] Hospital triage strip updates live for incoming bridge calls and incidents without manual refresh.
- [ ] Appointment booking completes in three decisions with visible and overridable earliest-slot default.
- [ ] Timezone reads in plain language.
- [ ] Video visit pre-join shows camera/mic state before entry.
- [ ] Video visit ended and missing-details states match the redesigned frames.

### Accessibility

- [ ] Keyboard-only completion works for login, booking, consent grant/revoke, and audit search on web.
- [ ] Visible focus appears at every keyboard step.
- [ ] Screen-reader labels exist on every interactive element in changed mobile screens.
- [ ] All text/background pairings pass AA in both themes.
- [ ] Touch targets are at least 44pt.
- [ ] Reduced-motion preference disables all non-essential animation.
- [ ] Patient-facing copy is verified at approximately 8th-grade reading level.

### Safety

- [ ] No PHI appears in console logs, toasts, notification previews, or error messages introduced by this work.
- [ ] AI-generated summaries always render with provenance and verify-first treatment.
- [ ] No path presents AI-generated summaries as plain facts.

### Validation Loop

- [ ] Web routes checked at 390x844, 768, 1280x720, and 1440x900 in both themes, with clean console.
- [x] Web production build passes.
- [ ] Web lint passes or any existing lint blocker is documented.
- [ ] Mobile tabs walked through Expo MCP on iOS and Android viewports.
- [ ] Mobile rotation/background/foreground behavior checked where supported.
- [ ] Register -> verify -> OTP -> portal flow exercised against local stack.
- [ ] QR consent scan -> grant -> revoke flow exercised against local stack.
- [ ] Book -> reschedule -> cancel appointment flow exercised against local stack.
- [ ] Schedule -> join -> end meeting flow exercised against local stack.
- [ ] Welfare check flow exercised across patient and hospital views.
- [ ] One GitHub issue exists for each remaining defect, with screenshot and Figma frame reference.

## Prototype Wiring Evidence

- 2026-08-12: Fixed Figma visible patient controls so prototype mode follows the intended core journeys without relying on hidden/page-level hotspots.
  - Web Patient `Button / Run device check` and `Button / Join visit` now route to `web/patient/meeting/prejoin`.
  - Mobile Patient `Button / Confirm booking` now routes to `mobile/patient/appointments/booked`.
  - Mobile Patient `Button / Join / device check` now routes to `mobile/patient/meeting/prejoin`.
- Re-read Figma destinations after the patch and updated `docs/design/peak-ux-verification.json` plus `docs/design/peak-ux-brief-audit.json`.

- 2026-08-12: Figma MCP approximate contrast scan checked 550 visible text nodes across five core Peak UX pages and found 0 AA contrast issues; full WCAG 2.2 AA remains open for keyboard, focus, touch-target, reduced-motion, and screen-reader verification. Evidence card: Figma node `31:2`.

- 2026-08-12: Wired patient navigation affordances that were visible but previously inert.
  - Web Patient: 36 non-active sidebar cards wired for Home/Care/Records/Privacy; reaction count is now 50.
  - Mobile Patient: 32 non-active bottom tabs wired for Home/Care/Records/Privacy; reaction count is now 47.
  - This gives QR consent and records-question journeys visible entries from patient Home/Care instead of requiring manual frame starts.
- 2026-08-12: Verified staff chain: web Home -> Triage incident -> Review patient summary -> Trace audit, and mobile Home -> Triage alert -> Trace audit / Review summary -> Open audit trail.

- 2026-08-12: Patched mobile patient target sizing and verified interactive node sizes.
  - Raised 45 Mobile Patient bottom tab instances from 62x39 to 62x45 by increasing existing vertical padding.
  - Re-scanned Web Patient, Web Hospital, Web Meeting, Mobile Patient, and Mobile Hospital: 112 interactive nodes, 0 WCAG 24x24 failures, 0 stricter 44x44 comfort failures.
  - Screenshot rendered for `mobile/patient/home` at 390x844; local visual viewer was blocked by the sandbox loopback error, so this remains measurement-backed rather than visually inspected in-tool.

- 2026-08-12: Added Figma accessibility metadata for interactive controls.
  - Wrote `zorba.a11y` metadata keys `label`, `focusOrder`, and `frame` to 112 interactive nodes across the five core Peak UX pages.
  - Re-scanned metadata coverage: 0 missing accessible labels and 0 missing focus-order entries.
  - Added `docs/design/peak-ux-accessibility-audit.json`; evidence card `31:2` includes note `42:2`.

- 2026-08-12: Refreshed the Current State baseline search and recorded that the hard decision-count gate remains unproved.
  - Searched repo docs plus Codex attachments/cache for a Current State Figma URL/key and `docs/design/current-state-dna.json`; found only prompt references, no baseline artifact.
  - `graphify query` for the current-state baseline exited 1 with no output.

- 2026-08-12: Fixed the web `PatientPicker` build-time a11y warning.
  - Added `aria-selected` to the `role="option"` result button at `web/src/components/PatientPicker.tsx:155`.
  - `npm run build` in `web/` passes and the previous `jsx-a11y/role-has-required-aria-props` warning did not recur.

- 2026-08-12: Rechecked the Figma evidence card through the available read-only connector tools.
  - Node `31:2` metadata includes `WCAG Note`, `Target Size Note`, and `A11y Metadata Note`.
  - Screenshot API rendered the evidence card at 760x518 and downloaded `/tmp/peak-ux-evidence-card-31-2.png` as 94,040 bytes.
  - Local visual inspection remains blocked by the sandbox loopback error, so this is render/download evidence rather than a visual QA pass.

- 2026-08-12: Added an explicit WCAG 2.2 AA gap audit.
  - `docs/design/peak-ux-wcag-gap-audit.json` separates proven evidence from partial/design-time evidence and missing runtime checks.
  - Strong evidence exists for core Figma text contrast, target size, and interactive metadata; full WCAG remains open for all-frame coverage plus runtime keyboard, focus-visible, screen-reader, and reduced-motion checks.

## Known Defects

- #43 Fix PatientPicker option aria-selected warning - fixed locally by adding `aria-selected` at `web/src/components/PatientPicker.tsx:155`; `npm run build` passes without the prior warning. GitHub issue can be closed after the commit is pushed.

## Current Gate

All staged PRs now link this issue. The next highest-impact gate is visual and live-flow verification: attach browser/Expo screenshots against Peak UX Figma frames, document any mismatch as a GitHub issue, and only then convert PRs from draft to review-ready.
