"use client";

import { useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { API_URL } from "../../constants";
import { APIEndpoints } from "../../contracts";
import { AlertCircle, CheckCircle2, Loader2, Mail } from "lucide-react";
import { DotPattern } from "../../components/magicui/dot-pattern";
import { MagicCard } from "../../components/magicui/magic-card";

type Status = "idle" | "loading" | "success" | "error" | "missing_token";

function VerifyEmailContent() {
  const searchParams = useSearchParams();
  const token = searchParams.get("token");
  const verifiedTokenRef = useRef<string | null>(null);

  const [status, setStatus] = useState<Status>("idle");
  const [errorMessage, setErrorMessage] = useState<string>("");

  const verify = useCallback(async (t: string) => {
    setStatus("loading");
    setErrorMessage("");

    try {
      const response = await fetch(
        `${API_URL}${APIEndpoints.PATIENT_REGISTER_VERIFY}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ token: t }),
        },
      );

      const data = await response.json().catch(() => ({}));

      if (response.ok) {
        setStatus("success");
      } else {
        setStatus("error");
        setErrorMessage(
          (data as { error?: { message?: string } })?.error?.message ||
            "Invalid or expired verification link.",
        );
      }
    } catch {
      setStatus("error");
      setErrorMessage("Network error. Please try again.");
    }
  }, []);

  useEffect(() => {
    if (!token) {
      setStatus("missing_token");
      return;
    }

    if (verifiedTokenRef.current === token) {
      return;
    }
    verifiedTokenRef.current = token;

    verify(token);
  }, [token, verify]);

  const showError = status === "error" || status === "missing_token";

  return (
    <main className="relative min-h-screen overflow-hidden bg-slate-950 text-slate-950">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(99,102,241,0.28),transparent_24%),radial-gradient(circle_at_top_right,rgba(249,115,22,0.18),transparent_18%),linear-gradient(180deg,#020617_0%,#0f172a_55%,#111827_100%)]" />
      <DotPattern
        glow
        className="text-indigo-300/30 [mask-image:radial-gradient(700px_circle_at_center,white,transparent)]"
      />
      <div className="relative flex min-h-screen flex-col items-center justify-center px-4 py-8">
        <MagicCard className="w-full max-w-md overflow-hidden rounded-3xl p-0" gradientFrom="#4f46e5" gradientTo="#f97316" gradientColor="rgba(79, 70, 229, 0.12)">
        <div className="relative w-full rounded-3xl overflow-hidden border border-white/60 bg-white/90 shadow-2xl shadow-slate-950/20 backdrop-blur-xl dark:border-slate-800 dark:bg-slate-950/90">
          
          {status === "loading" && (
            <div className="absolute top-0 left-0 w-full h-1 bg-indigo-500 animate-pulse" />
          )}
          {status === "success" && (
            <div className="absolute top-0 left-0 w-full h-1 bg-emerald-500" />
          )}
          {showError && (
            <div className="absolute top-0 left-0 w-full h-1 bg-rose-500" />
          )}

          <div className="px-8 py-10 flex flex-col items-center text-center">
            {/* Status Icons */}
            {status === "loading" && (
              <div className="mb-6 inline-flex h-16 w-16 items-center justify-center rounded-3xl bg-indigo-50 text-indigo-600 ring-1 ring-indigo-100">
                <Loader2 className="h-8 w-8 animate-spin" />
              </div>
            )}

            {status === "success" && (
              <div className="mb-6 inline-flex h-16 w-16 items-center justify-center rounded-3xl bg-emerald-50 text-emerald-600 ring-1 ring-emerald-100">
                <CheckCircle2 className="h-8 w-8" />
              </div>
            )}

            {showError && (
              <div className="mb-6 inline-flex h-16 w-16 items-center justify-center rounded-3xl bg-rose-50 text-rose-600 ring-1 ring-rose-100">
                <AlertCircle className="h-8 w-8" />
              </div>
            )}

            {status === "idle" && (
              <div className="mb-6 inline-flex h-16 w-16 items-center justify-center rounded-3xl bg-slate-50 text-slate-400">
                <Mail className="h-8 w-8" />
              </div>
            )}

            {/* Dynamic Status Blocks */}
            {status === "loading" && (
              <>
                <h1 className="text-2xl font-black text-slate-950 tracking-tight mb-2">
                  Verifying Email
                </h1>
                <p className="text-slate-500 text-sm leading-relaxed mb-4">
                  Please hold on while we verify your patient activation token...
                </p>
              </>
            )}

            {status === "success" && (
              <>
                <h1 className="text-2xl font-black text-slate-950 tracking-tight mb-2">
                  Account Verified!
                </h1>
                <p className="text-slate-500 text-sm leading-relaxed mb-8">
                  Your email verification is complete. You can now log in and manage your Zorba Health profile.
                </p>
                <Link
                  href="/login/patient"
                  className="w-full inline-flex items-center justify-center h-12 rounded-xl bg-gradient-to-r from-indigo-600 to-orange-500 text-white font-bold shadow-orange transition-all duration-200 hover:-translate-y-0.5"
                >
                  Continue to Sign In
                </Link>
              </>
            )}

            {showError && (
              <>
                <h1 className="text-2xl font-black text-slate-950 tracking-tight mb-2">
                  Verification Failed
                </h1>
                <p className="text-slate-500 text-sm leading-relaxed mb-6">
                  {status === "missing_token"
                    ? "This verification link is invalid or is missing a token. Please click the exact link from your email."
                    : "The email verification link has expired or is invalid. Please try registering again."}
                </p>
                {errorMessage && (
                  <div className="mb-6 w-full rounded-xl bg-slate-50 p-3 text-xs text-slate-400 font-mono break-all border border-slate-100">
                    {errorMessage}
                  </div>
                )}
                <div className="w-full flex flex-col sm:flex-row gap-3">
                  <Link
                    href="/register/patient"
                    className="flex-1 inline-flex items-center justify-center h-11 rounded-xl border border-slate-200 bg-white text-slate-700 font-bold text-sm hover:bg-slate-50 transition-colors"
                  >
                    Register Again
                  </Link>
                  <Link
                    href="/login/patient"
                    className="flex-1 inline-flex items-center justify-center h-11 rounded-xl bg-indigo-600 hover:bg-indigo-700 text-white font-bold text-sm shadow-glow transition-all duration-200"
                  >
                    Sign In
                  </Link>
                </div>
              </>
            )}
          </div>
        </div>
        </MagicCard>
      </div>
    </main>
  );
}

export default function VerifyEmailPage() {
  return (
    <Suspense
      fallback={
        <main className="min-h-screen mesh-bg text-slate-950">
          <div className="flex flex-col items-center justify-center min-h-screen px-4 py-8">
            <div className="glass-card w-full max-w-md rounded-3xl overflow-hidden text-center py-12">
              <h1 className="text-2xl font-black text-slate-950 tracking-tight mb-2">
                Email Verification
              </h1>
              <p className="text-slate-500 text-sm">Preparing verification environment...</p>
            </div>
          </div>
        </main>
      }
    >
      <VerifyEmailContent />
    </Suspense>
  );
}
