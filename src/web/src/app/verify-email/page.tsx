"use client";

import { useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { API_URL } from "../../constants";
import { APIEndpoints } from "../../contracts";

type Status = "idle" | "loading" | "success" | "error" | "missing_token";

function VerifyEmailContent() {
  const searchParams = useSearchParams();
  const token = searchParams.get("token");

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
    verify(token);
  }, [token, verify]);

  const showError = status === "error" || status === "missing_token";

  return (
    <main className="min-h-screen bg-slate-100">
      <div className="flex flex-col items-center justify-center min-h-screen px-4">
        <div className="w-full max-w-3xl rounded-2xl bg-white shadow-xl overflow-hidden">
          <div className="h-2 bg-gradient-to-r from-emerald-500 to-emerald-600" />

          <div className="px-8 py-10 flex flex-col items-center text-center">
            <div className="mb-6 inline-flex h-20 w-20 items-center justify-center rounded-full bg-emerald-50">
              <span className="text-4xl" aria-hidden="true">
                📧
              </span>
            </div>

            {status === "loading" && (
              <>
                <h1 className="text-2xl font-bold text-slate-900 mb-3">
                  Verifying your email…
                </h1>
                <p className="text-slate-600 text-sm mb-4">
                  Sit tight while we confirm your email address.
                </p>
                <div className="mt-2 flex justify-center">
                  <span
                    className="h-5 w-5 border-2 border-emerald-500 border-t-transparent rounded-full animate-spin"
                    aria-hidden="true"
                  />
                </div>
              </>
            )}

            {status === "success" && (
              <>
                <h1 className="text-2xl font-bold text-slate-900 mb-3">
                  You&apos;re all set!
                </h1>
                <p className="text-slate-600 text-sm mb-6">
                  Your email has been verified. You can now sign in and start
                  using Zorba Health.
                </p>
                <Link
                  href="/login/patient"
                  className="inline-flex items-center justify-center rounded-full bg-emerald-500 px-6 py-3 text-sm font-semibold text-white shadow hover:bg-emerald-600 focus:outline-none focus:ring-2 focus:ring-emerald-500 focus:ring-offset-2"
                >
                  Go to sign in
                </Link>
              </>
            )}

            {showError && (
              <>
                <h1 className="text-2xl font-bold text-slate-900 mb-3">
                  Sorry, there was a problem.
                </h1>
                <p className="text-slate-600 text-sm mb-4">
                  {status === "missing_token"
                    ? "This verification link is invalid or missing a token. Please open the link directly from your verification email or request a new one."
                    : "We couldn’t verify your email. The link may be invalid or expired. Please try again or request a new verification email."}
                </p>
                {errorMessage && (
                  <p className="text-xs text-slate-400 mb-4">{errorMessage}</p>
                )}
                <div className="flex flex-wrap justify-center gap-3 mt-2">
                  <Link
                    href="/register/patient"
                    className="inline-flex items-center justify-center rounded-full border border-slate-300 bg-white px-5 py-2.5 text-sm font-medium text-slate-700 hover:bg-slate-50"
                  >
                    Register again
                  </Link>
                  <Link
                    href="/login/patient"
                    className="inline-flex items-center justify-center rounded-full bg-emerald-500 px-5 py-2.5 text-sm font-medium text-white hover:bg-emerald-600"
                  >
                    Sign in
                  </Link>
                </div>
              </>
            )}
          </div>
        </div>
      </div>
    </main>
  );
}

export default function VerifyEmailPage() {
  return (
    <Suspense
      fallback={
        <main className="min-h-screen bg-slate-100">
          <div className="flex flex-col items-center justify-center min-h-screen px-4">
            <div className="w-full max-w-3xl rounded-2xl bg-white shadow-xl overflow-hidden">
              <div className="h-2 bg-gradient-to-r from-emerald-500 to-emerald-600" />
              <div className="px-8 py-10 text-center">
                <h1 className="text-2xl font-bold text-slate-900 mb-3">
                  Email verification
                </h1>
                <p className="text-slate-600 text-sm">
                  Loading verification details…
                </p>
              </div>
            </div>
          </div>
        </main>
      }
    >
      <VerifyEmailContent />
    </Suspense>
  );
}
