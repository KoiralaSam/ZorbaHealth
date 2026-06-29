import Link from "next/link";
import { MailCheck, ShieldCheck } from "lucide-react";
import { DotPattern } from "../../../../components/magicui/dot-pattern";
import { MagicCard } from "../../../../components/magicui/magic-card";

export default function VerifyEmailPage() {
  return (
    <main className="relative min-h-screen overflow-hidden px-4 py-8 text-slate-950 bg-slate-950">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(99,102,241,0.28),transparent_24%),radial-gradient(circle_at_top_right,rgba(249,115,22,0.18),transparent_18%),linear-gradient(180deg,#020617_0%,#0f172a_55%,#111827_100%)]" />
      <DotPattern
        glow
        className="text-indigo-300/30 [mask-image:radial-gradient(700px_circle_at_center,white,transparent)]"
      />
      <div className="relative flex min-h-[calc(100vh-4rem)] items-center justify-center">
        <MagicCard className="w-full max-w-md overflow-hidden rounded-3xl p-0" gradientFrom="#4f46e5" gradientTo="#f97316" gradientColor="rgba(79, 70, 229, 0.12)">
        <section className="relative w-full rounded-3xl border border-white/60 bg-white/90 p-8 text-center shadow-2xl shadow-slate-950/20 backdrop-blur-xl dark:border-slate-800 dark:bg-slate-950/90">
          <div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-3xl bg-indigo-50 text-indigo-600 ring-1 ring-indigo-100">
            <MailCheck className="h-8 w-8" />
          </div>
          <p className="text-xs font-black uppercase tracking-wide text-indigo-600">
            Step 3 of 3
          </p>
          <h1 className="mt-2 text-3xl font-black tracking-tight">
            Check your email
          </h1>
          <p className="mt-3 text-sm leading-7 text-slate-600">
            We sent a verification link to your email address. Open it to
            activate your patient account and finish registration.
          </p>
          <div className="mt-6 rounded-2xl border border-indigo-100 bg-indigo-50/70 p-4 text-left">
            <div className="flex items-start gap-3">
              <ShieldCheck className="mt-0.5 h-5 w-5 shrink-0 text-indigo-600" />
              <p className="text-sm font-semibold leading-6 text-indigo-950">
                If you do not see the message, check spam or junk mail before
                requesting a new registration attempt.
              </p>
            </div>
          </div>
          <Link
            href="/login/patient"
            className="mt-7 inline-flex h-12 w-full items-center justify-center rounded-xl bg-gradient-to-r from-indigo-600 to-orange-500 text-sm font-bold text-white shadow-orange transition-all hover:-translate-y-0.5"
          >
            Continue to Sign In
          </Link>
        </section>
        </MagicCard>
      </div>
    </main>
  );
}
