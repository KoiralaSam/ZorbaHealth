"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "../components/ui/button";
import {
  ArrowRight,
  BellRing,
  Building2,
  Globe2,
  Languages,
  MapPinned,
  Mic2,
  ShieldCheck,
  Sparkles,
  UserRound,
} from "lucide-react";
import { AppHeader } from "../components/layout/app-header";
import { cn } from "../lib/utils";
import { AnimatedList } from "../components/magicui/animated-list";
import { AnimatedShinyText } from "../components/magicui/animated-shiny-text";
import { AuroraText } from "../components/magicui/aurora-text";
import { BentoCard, BentoGrid } from "../components/magicui/bento-grid";
import { DotPattern } from "../components/magicui/dot-pattern";
import { DottedMap, type Marker } from "../components/magicui/dotted-map";
import { FlickeringGrid } from "../components/magicui/flickering-grid";
import { Marquee } from "../components/magicui/marquee";
import { NumberTicker } from "../components/magicui/number-ticker";
import { ShimmerButton } from "../components/magicui/shimmer-button";

const transcriptSnippets = [
  {
    title: "Triage call",
    body: "Patient symptoms summarized and routed to clinical review.",
  },
  {
    title: "Consent verified",
    body: "Record access approved before the assistant answered.",
  },
  {
    title: "Medication question",
    body: "Grounded response returned with source references.",
  },
  {
    title: "GPS alert",
    body: "Safety location streamed into the patient emergency workflow.",
  },
];

const liveAlerts = [
  {
    icon: "🩺",
    title: "Voice triage escalated",
    body: "Chest pain symptoms flagged for the care team.",
    time: "Now",
  },
  {
    icon: "🔐",
    title: "Consent updated",
    body: "Patient granted limited records access for a summary request.",
    time: "2m",
  },
  {
    icon: "📝",
    title: "Audit event written",
    body: "Every hospital lookup was recorded with actor + timestamp.",
    time: "5m",
  },
  {
    icon: "🌍",
    title: "Translator connected",
    body: "Bilingual call support enabled during the live session.",
    time: "8m",
  },
];

const featureCards = [
  {
    Icon: Mic2,
    name: "Voice AI that sounds clinical, not robotic",
    description:
      "Use natural-language calling flows for triage, refill questions, record Q&A, and follow-up coordination.",
    href: "/login/patient",
    cta: "Open patient experience",
    className: "col-span-3 lg:col-span-2",
    background: (
      <div className="absolute inset-0 overflow-hidden">
        <Marquee pauseOnHover className="[--duration:28s] pt-8">
          {transcriptSnippets.map((snippet) => (
            <figure
              key={snippet.title}
              className="w-64 rounded-3xl border border-indigo-200/60 bg-white/85 p-4 shadow-lg shadow-indigo-100/60 backdrop-blur dark:border-white/10 dark:bg-white/5 dark:shadow-none"
            >
              <p className="text-xs font-black uppercase tracking-[0.28em] text-indigo-500">
                {snippet.title}
              </p>
              <p className="mt-3 text-sm leading-6 text-slate-600 dark:text-slate-300">
                {snippet.body}
              </p>
            </figure>
          ))}
        </Marquee>
        <div className="pointer-events-none absolute inset-y-0 left-0 w-24 bg-gradient-to-r from-background to-transparent" />
        <div className="pointer-events-none absolute inset-y-0 right-0 w-24 bg-gradient-to-l from-background to-transparent" />
      </div>
    ),
  },
  {
    Icon: ShieldCheck,
    name: "Consent and audit are first-class UX",
    description:
      "Patients can understand what they are approving, and hospitals can trace every retrieval and alert.",
    href: "/login/hospital",
    cta: "Review the console",
    className: "col-span-3 lg:col-span-1",
    background: (
      <div className="absolute inset-x-0 top-6 px-4">
        <AnimatedList delay={1200} className="gap-3">
          {liveAlerts.slice(0, 3).map((alert) => (
            <div
              key={alert.title}
              className="rounded-[1.4rem] border border-slate-200/70 bg-white/90 p-3 shadow-lg shadow-slate-200/70 dark:border-white/10 dark:bg-white/5 dark:shadow-none"
            >
              <div className="flex items-start gap-3">
                <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-indigo-50 text-lg dark:bg-white/10">
                  {alert.icon}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <p className="text-[13px] font-black text-slate-900 dark:text-white">
                      {alert.title}
                    </p>
                    <span className="text-xs font-semibold text-slate-400">
                      {alert.time}
                    </span>
                  </div>
                  <p className="mt-1 text-xs leading-5 text-slate-500 dark:text-slate-300">
                    {alert.body}
                  </p>
                </div>
              </div>
            </div>
          ))}
        </AnimatedList>
      </div>
    ),
  },
  {
    Icon: Globe2,
    name: "Care coordination stays visible across locations",
    description:
      "Bridge calls, multilingual support, and safety workflows feel connected across the patient journey.",
    href: "/register/hospital",
    cta: "Set up a hospital",
    className: "col-span-3 lg:col-span-1",
    background: (
      <div className="absolute inset-0 px-6 py-8">
        <div className="absolute inset-6 rounded-[2rem] bg-gradient-to-br from-indigo-100/70 via-transparent to-orange-100/80 dark:from-indigo-500/10 dark:to-orange-500/10" />
        <DottedMap<Marker>
          className="absolute inset-6 text-indigo-300/80 dark:text-indigo-200/30"
          markers={[
            { lat: 40.7128, lng: -74.006, size: 0.8, pulse: true },
            { lat: 51.5072, lng: -0.1276, size: 0.8, pulse: true },
            { lat: 27.7172, lng: 85.324, size: 0.8, pulse: true },
          ]}
          markerColor="#f97316"
          dotColor="currentColor"
          dotRadius={0.22}
          pulse
        />
        <div className="absolute left-6 top-6 rounded-full border border-white/70 bg-white/80 px-3 py-2 backdrop-blur dark:border-white/10 dark:bg-slate-950/70">
          <div className="flex items-center gap-2 text-[11px] font-black uppercase tracking-[0.24em] text-slate-500">
            <MapPinned className="h-3.5 w-3.5 text-orange-500" />
            Active support regions
          </div>
        </div>
      </div>
    ),
  },
  {
    Icon: Languages,
    name: "Interpretation is part of the workflow",
    description:
      "Switch language support on while the session stays routed, captioned, and documented.",
    href: "/login/hospital",
    cta: "See translation workflows",
    className: "col-span-3 lg:col-span-2",
    background: (
      <div className="absolute inset-0 flex items-center justify-center overflow-hidden px-6">
        <div className="grid w-full max-w-xl gap-4 sm:grid-cols-[1.1fr_0.9fr]">
          <div className="rounded-[1.75rem] border border-slate-200/70 bg-white/90 p-5 shadow-lg shadow-slate-200/70 dark:border-white/10 dark:bg-white/5 dark:shadow-none">
            <p className="text-xs font-black uppercase tracking-[0.24em] text-indigo-500">
              Live Interpretation
            </p>
            <p className="mt-3 text-sm leading-6 text-slate-600 dark:text-slate-300">
              Switch staff audio between translated and original speech without
              interrupting the consultation.
            </p>
            <div className="mt-4 flex flex-wrap gap-2">
              {["EN", "ES", "NP", "AR"].map((language) => (
                <span
                  key={language}
                  className="inline-flex h-10 min-w-10 items-center justify-center rounded-full border border-indigo-200/70 bg-indigo-50 px-3 text-sm font-black text-indigo-700 dark:border-indigo-400/20 dark:bg-indigo-500/10 dark:text-indigo-200"
                >
                  {language}
                </span>
              ))}
            </div>
          </div>
          <div className="rounded-[1.75rem] border border-slate-200/70 bg-white/90 p-5 shadow-lg shadow-slate-200/70 dark:border-white/10 dark:bg-white/5 dark:shadow-none">
            <p className="text-xs font-black uppercase tracking-[0.24em] text-orange-500">
              Workflow Notes
            </p>
            <div className="mt-3 space-y-3 text-sm text-slate-600 dark:text-slate-300">
              <div className="rounded-2xl bg-slate-50 px-4 py-3 dark:bg-white/5">
                Auto-detect patient language when the call begins.
              </div>
              <div className="rounded-2xl bg-slate-50 px-4 py-3 dark:bg-white/5">
                Keep captions visible for clinicians during escalation.
              </div>
              <div className="rounded-2xl bg-slate-50 px-4 py-3 dark:bg-white/5">
                Save translation preferences directly into the bridge session.
              </div>
            </div>
          </div>
        </div>
      </div>
    ),
  },
];

function SectionEyebrow({
  children,
  className,
  textClassName,
}: {
  children: React.ReactNode;
  className?: string;
  textClassName?: string;
}) {
  return (
    <div
      className={cn(
        "inline-flex items-center gap-2 rounded-full border border-indigo-200/60 bg-white/70 px-4 py-2 backdrop-blur dark:border-white/10 dark:bg-white/5",
        className,
      )}
    >
      <Sparkles className="h-4 w-4 text-orange-500" />
      <AnimatedShinyText
        className={cn(
          "inline-flex text-xs font-black uppercase tracking-[0.28em] text-slate-600 dark:text-slate-200",
          textClassName,
        )}
      >
        {children}
      </AnimatedShinyText>
    </div>
  );
}

export default function Home() {
  const router = useRouter();

  return (
    <main
      id="main-content"
      className="min-h-screen overflow-x-hidden bg-slate-950 text-slate-950 dark:text-slate-50"
    >
      <AppHeader
        links={[
          { href: "/login/patient", label: "Patient Portal" },
          { href: "/login/hospital", label: "Hospital Portal" },
        ]}
        className="border-white/10 bg-slate-950/70 text-white dark:bg-slate-950/70"
      />

      <section className="relative overflow-hidden px-5 pb-14 pt-12 md:pt-20">
        <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(99,102,241,0.3),transparent_28%),radial-gradient(circle_at_top_right,rgba(249,115,22,0.22),transparent_22%),radial-gradient(circle_at_bottom,rgba(14,165,233,0.14),transparent_26%),linear-gradient(180deg,#020617_0%,#0f172a_52%,#111827_100%)]" />
        <DotPattern
          glow
          className="text-indigo-300/35 [mask-image:radial-gradient(520px_circle_at_center,white,transparent)]"
        />
        <div className="relative mx-auto grid max-w-7xl gap-10 lg:grid-cols-[1.12fr_0.88fr] lg:items-center">
          <div className="animate-fade-in-up">
            <SectionEyebrow
              className="border-white/10 bg-white/10"
              textClassName="text-white/80 dark:text-white/80"
            >
              Clinical voice intelligence
            </SectionEyebrow>
            <h1 className="mt-8 max-w-5xl text-5xl font-black leading-[0.98] tracking-tight text-white md:text-7xl">
              Healthcare access that feels{" "}
              <AuroraText className="font-black">guided, calm, and unmistakably modern.</AuroraText>
            </h1>
            <p className="mt-6 max-w-2xl text-base leading-8 text-slate-200 md:text-lg">
              Zorba Health brings voice AI, consent-aware records, multilingual
              support, and hospital-grade auditability into one experience that
              feels premium for both patients and care teams.
            </p>
            <div className="mt-10 flex flex-col gap-4 sm:flex-row">
              <ShimmerButton
                className="h-14 px-7 shadow-2xl shadow-indigo-950/50"
                onClick={() => router.push("/register/patient")}
              >
                <span className="inline-flex items-center gap-2 text-sm font-black tracking-[0.22em] uppercase">
                  Start patient onboarding
                  <ArrowRight className="h-4 w-4" />
                </span>
              </ShimmerButton>
              <Button
                variant="outline"
                size="lg"
                asChild
                className="h-14 rounded-full border-white/15 bg-white/5 px-7 text-white hover:bg-white/10"
              >
                <Link href="/register/hospital">Register Hospital</Link>
              </Button>
            </div>

            <div className="mt-10 grid gap-4 sm:grid-cols-3">
              {[
                {
                  label: "patient voice sessions",
                  value: 24,
                  suffix: "/7",
                  tone: "text-indigo-200",
                },
                {
                  label: "audit-backed workflows",
                  value: 100,
                  suffix: "%",
                  tone: "text-orange-200",
                },
                {
                  label: "care teams kept in sync",
                  value: 12,
                  suffix: "x",
                  tone: "text-sky-200",
                },
              ].map((stat) => (
                <div
                  key={stat.label}
                  className="rounded-[1.75rem] border border-white/10 bg-white/10 p-5 backdrop-blur"
                >
                  <div className={cn("text-3xl font-black", stat.tone)}>
                    <NumberTicker value={stat.value} className={stat.tone} />
                    <span>{stat.suffix}</span>
                  </div>
                  <p className="mt-2 text-sm leading-6 text-slate-200">
                    {stat.label}
                  </p>
                </div>
              ))}
            </div>
          </div>

          <div className="relative overflow-hidden rounded-[2rem] border border-white/10 bg-white/5 p-6 backdrop-blur">
            <FlickeringGrid
              className="absolute inset-0 [mask-image:radial-gradient(340px_circle_at_center,white,transparent)]"
              squareSize={4}
              gridGap={7}
              color="#94a3b8"
              maxOpacity={0.45}
              flickerChance={0.12}
            />
            <div className="relative rounded-[1.75rem] border border-white/10 bg-slate-950/65 p-6 text-white">
              <p className="text-xs font-black uppercase tracking-[0.3em] text-indigo-200">
                Patient + hospital operating surface
              </p>
              <h2 className="mt-3 text-3xl font-black leading-tight">
                Built for voice-driven care, escalations, and human review.
              </h2>
              <div className="mt-6 grid gap-3">
                {[
                  "Voice assistant guided with record-aware answers",
                  "Consent checkpoints surfaced before data moves",
                  "Escalations, incidents, and summaries visible in one console",
                ].map((item) => (
                  <div
                    key={item}
                    className="rounded-2xl border border-white/10 bg-white/10 px-4 py-3 text-sm font-semibold text-slate-100"
                  >
                    {item}
                  </div>
                ))}
              </div>
              <div className="mt-6 grid gap-3 sm:grid-cols-2">
                <Link
                  href="/login/patient"
                  className="rounded-[1.5rem] border border-white/10 bg-white/10 p-4 transition hover:bg-white/15"
                >
                  <UserRound className="h-6 w-6 text-indigo-200" />
                  <p className="mt-4 text-lg font-black">Patient Portal</p>
                  <p className="mt-1 text-sm leading-6 text-slate-300">
                    Consents, records, calls, GPS safety.
                  </p>
                </Link>
                <Link
                  href="/login/hospital"
                  className="rounded-[1.5rem] border border-white/10 bg-white/10 p-4 transition hover:bg-white/15"
                >
                  <Building2 className="h-6 w-6 text-orange-200" />
                  <p className="mt-4 text-lg font-black">Hospital Console</p>
                  <p className="mt-1 text-sm leading-6 text-slate-300">
                    Summaries, incidents, audit search, live care signals.
                  </p>
                </Link>
              </div>
            </div>
          </div>
        </div>
      </section>

      <section className="relative bg-slate-50 px-5 py-16 text-slate-950 dark:bg-slate-950 dark:text-slate-50">
        <DotPattern className="text-indigo-300/40 [mask-image:radial-gradient(640px_circle_at_center,white,transparent)]" />
        <div className="relative mx-auto max-w-7xl">
          <div className="mb-8 max-w-3xl">
            <SectionEyebrow>Connected product surfaces</SectionEyebrow>
            <h2 className="mt-6 text-4xl font-black tracking-tight md:text-5xl">
              A redesign that shows the product is orchestrating care, not just listing screens.
            </h2>
            <p className="mt-4 text-base leading-8 text-slate-600 dark:text-slate-300">
              Every panel below comes from the new Magic UI-driven system: more motion,
              more signal, and a more premium healthcare feel.
            </p>
          </div>
          <BentoGrid className="lg:auto-rows-[24rem]">
            {featureCards.map((feature) => (
              <BentoCard key={feature.name} {...feature} />
            ))}
          </BentoGrid>
        </div>
      </section>

      <footer className="border-t border-slate-200 bg-white px-5 py-6 dark:border-slate-800 dark:bg-slate-950">
        <div className="mx-auto flex max-w-7xl flex-col gap-4 text-sm font-semibold text-slate-500 sm:flex-row sm:items-center sm:justify-between">
          <span>Designed for premium, traceable healthcare interactions.</span>
          <div className="flex flex-wrap gap-2">
            {["Voice AI", "Consent aware", "Audit ready", "Live safety"].map((item) => (
              <span
                key={item}
                className="inline-flex items-center gap-2 rounded-full border border-slate-200 bg-slate-50 px-3 py-1.5 dark:border-slate-800 dark:bg-slate-900"
              >
                <BellRing className="h-3.5 w-3.5 text-indigo-500" />
                {item}
              </span>
            ))}
          </div>
        </div>
      </footer>
    </main>
  );
}
