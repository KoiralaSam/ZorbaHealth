"use client";

import { Suspense, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Button } from "../../../../components/ui/button";
import {
  ArrowLeft,
  Clock,
  Phone,
  ShieldAlert,
  ShieldCheck,
  Smartphone,
} from "lucide-react";
import { API_URL } from "../../../../constants";
import { DotPattern } from "../../../../components/magicui/dot-pattern";
import { MagicCard } from "../../../../components/magicui/magic-card";
import {
  APIEndpoints,
  HTTPPatientRegisterRequest,
  HTTPPatientVerifyOTPRequest,
  HTTPPatientVerifyOTPResponse,
} from "../../../../contracts";

function PatientVerifyOTPContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const phoneFromQuery = useMemo(
    () => searchParams.get("phone") || "",
    [searchParams]
  );

  const [phoneNumber, setPhoneNumber] = useState(phoneFromQuery);
  const [otp, setOtp] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [isResending, setIsResending] = useState(false);
  const [resendSecondsLeft, setResendSecondsLeft] = useState(15);
  const [registrationRequest, setRegistrationRequest] =
    useState<HTTPPatientRegisterRequest | null>(null);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [statusType, setStatusType] = useState<"error" | "success" | "info">(
    "info"
  );

  useEffect(() => {
    if (typeof window === "undefined") return;

    try {
      const saved = window.sessionStorage.getItem("patientRegistration");
      if (!saved) return;

      const parsed = JSON.parse(saved) as HTTPPatientRegisterRequest;
      setRegistrationRequest(parsed);

      if (!phoneFromQuery && parsed.phone_number) {
        setPhoneNumber(parsed.phone_number);
      }
    } catch {
      // ignore invalid saved data
    }
  }, [phoneFromQuery]);

  useEffect(() => {
    if (resendSecondsLeft <= 0) return;

    const id = window.setInterval(() => {
      setResendSecondsLeft((prev) => {
        if (prev <= 1) {
          window.clearInterval(id);
          return 0;
        }
        return prev - 1;
      });
    }, 1000);

    return () => window.clearInterval(id);
  }, [resendSecondsLeft]);

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    setStatusMessage(null);

    if (!phoneNumber || !otp) {
      setStatusType("error");
      setStatusMessage("Phone number and OTP are required.");
      return;
    }

    setIsLoading(true);
    try {
      const payload: HTTPPatientVerifyOTPRequest = {
        phone_number: phoneNumber,
        otp,
      };

      const response = await fetch(
        `${API_URL}${APIEndpoints.PATIENT_REGISTER_VERIFY_OTP}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        }
      );

      const data: HTTPPatientVerifyOTPResponse = await response
        .json()
        .catch(() => ({}));

      if (response.ok) {
        router.push("/register/patient/verify");
      } else {
        setStatusType("error");
        setStatusMessage(
          data.error?.message ||
            "OTP verification failed. Please check the code and try again."
        );
      }
    } catch {
      setStatusType("error");
      setStatusMessage("Network error. Please try again.");
    } finally {
      setIsLoading(false);
    }
  };

  const handleResendOTP = async () => {
    if (!registrationRequest) {
      setStatusType("error");
      setStatusMessage(
        "We couldn't find your registration details. Please go back and start registration again."
      );
      return;
    }

    if (!phoneNumber) {
      setStatusType("error");
      setStatusMessage(
        "Please enter a valid phone number before resending OTP."
      );
      return;
    }

    const updatedRequest: HTTPPatientRegisterRequest = {
      ...registrationRequest,
      phone_number: phoneNumber,
    };

    if (typeof window !== "undefined") {
      window.sessionStorage.setItem(
        "patientRegistration",
        JSON.stringify(updatedRequest)
      );
    }

    setIsResending(true);
    try {
      const response = await fetch(`${API_URL}${APIEndpoints.PATIENT_REGISTER}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(updatedRequest),
      });

      if (response.ok) {
        setStatusType("success");
        setStatusMessage("A new OTP has been sent to your phone number.");
        setResendSecondsLeft(15);
      } else {
        const data = await response.json().catch(() => ({}));
        setStatusType("error");
        setStatusMessage(
          data?.error?.message ||
            "Failed to resend OTP. Please check your number and try again."
        );
      }
    } catch {
      setStatusType("error");
      setStatusMessage("Network error. Please try again.");
    } finally {
      setIsResending(false);
    }
  };

  const formattedCountdown = useMemo(() => {
    const minutes = Math.floor(resendSecondsLeft / 60)
      .toString()
      .padStart(1, "0");
    const seconds = (resendSecondsLeft % 60).toString().padStart(2, "0");
    return `${minutes}:${seconds}`;
  }, [resendSecondsLeft]);

  const countdownPercent = Math.max(0, Math.min(100, (resendSecondsLeft / 15) * 100));

  return (
    <main className="relative min-h-screen overflow-hidden bg-slate-950 text-slate-950">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(99,102,241,0.28),transparent_24%),radial-gradient(circle_at_top_right,rgba(249,115,22,0.18),transparent_18%),linear-gradient(180deg,#020617_0%,#0f172a_55%,#111827_100%)]" />
      <DotPattern
        glow
        className="text-indigo-300/30 [mask-image:radial-gradient(700px_circle_at_center,white,transparent)]"
      />
      <div className="relative flex min-h-screen flex-col items-center justify-center gap-6 px-4 py-8">
        <MagicCard
          className="max-w-md w-full overflow-hidden rounded-3xl p-0"
          gradientFrom="#4f46e5"
          gradientTo="#f97316"
          gradientColor="rgba(79, 70, 229, 0.12)"
        >
        <div className="relative w-full rounded-3xl border border-white/60 bg-white/90 p-8 shadow-2xl shadow-slate-950/20 backdrop-blur-xl dark:border-slate-800 dark:bg-slate-950/90">
          <button
            onClick={() => router.push("/register/patient")}
            className="inline-flex items-center gap-1.5 text-slate-500 hover:text-slate-900 mb-6 text-sm font-bold transition-colors"
            type="button"
          >
            <ArrowLeft className="h-4 w-4" /> Back
          </button>

          <div className="mb-5 flex h-16 w-16 items-center justify-center rounded-3xl bg-indigo-50 text-indigo-600 ring-1 ring-indigo-100">
            <Smartphone className="h-7 w-7" />
          </div>
          <p className="text-xs font-black uppercase tracking-wide text-indigo-600">Step 2 of 3</p>
          <h2 className="mt-2 text-3xl font-black text-slate-950 tracking-tight mb-2">
            Verify Phone Number
          </h2>
          <p className="text-slate-500 text-sm leading-relaxed mb-6">
            Enter the one-time code (OTP) sent to your mobile phone. You may edit your number below if it was entered incorrectly.
          </p>

          {statusMessage && (
            <div
              className={`mb-6 rounded-xl border p-4 text-sm flex items-start gap-2 ${
                statusType === "error"
                  ? "border-rose-200 bg-rose-50/70 text-rose-700"
                  : statusType === "success"
                  ? "border-emerald-200 bg-emerald-50/70 text-emerald-700"
                  : "border-indigo-200 bg-indigo-50/70 text-indigo-700"
              }`}
            >
              {statusType === "error" ? (
                <ShieldAlert className="h-5 w-5 text-rose-600 shrink-0 mt-0.5" />
              ) : (
                <ShieldCheck className="h-5 w-5 text-emerald-600 shrink-0 mt-0.5" />
              )}
              <span>{statusMessage}</span>
            </div>
          )}

          <form onSubmit={handleVerify} className="space-y-5">
            <div>
              <label
                htmlFor="phone"
                className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2"
              >
                Phone Number
              </label>
              <div className="relative">
                <Phone className="absolute left-3 top-3.5 h-5 w-5 text-slate-400" />
                <input
                  id="phone"
                  type="tel"
                  value={phoneNumber}
                  onChange={(e) => {
                    const nextPhone = e.target.value;
                    setPhoneNumber(nextPhone);
                    setRegistrationRequest((prev) =>
                      prev
                        ? {
                            ...prev,
                            phone_number: nextPhone,
                          }
                        : prev
                    );
                  }}
                  placeholder="+15551234567"
                  className="w-full pl-10 pr-4 py-3 border border-slate-200 rounded-xl focus:ring-4 focus:ring-indigo-100 focus:border-indigo-400 text-sm transition-all placeholder:text-slate-400 outline-none"
                  required
                />
              </div>
            </div>

            <div>
              <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                One-Time Code (OTP)
              </label>
              <div className="grid grid-cols-6 gap-2">
                {Array.from({ length: 6 }).map((_, index) => (
                  <input
                    key={index}
                    aria-label={`OTP digit ${index + 1}`}
                    inputMode="numeric"
                    maxLength={1}
                    value={otp[index] ?? ""}
                    onChange={(e) => {
                      const digit = e.target.value.replace(/\D/g, "").slice(-1);
                      const chars = otp.padEnd(6, " ").split("");
                      chars[index] = digit || " ";
                      setOtp(chars.join("").replace(/\s/g, "").slice(0, 6));
                    }}
                    className="h-12 rounded-xl border border-slate-200 bg-white text-center text-xl font-black text-slate-950 outline-none transition-all focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
                  />
                ))}
              </div>
            </div>

            <Button
              type="submit"
              variant="healthcare"
              className="w-full text-base h-12"
              disabled={isLoading}
            >
              {isLoading ? "Verifying..." : "Verify OTP"}
            </Button>

            <div className="pt-6 border-t border-slate-200/70">
              <div className="mb-3 flex items-center justify-between gap-3">
                <p className="text-xs text-slate-500 flex items-center gap-1 font-semibold">
                  <Clock className="h-3.5 w-3.5" />
                  Cooldown timer
                </p>
                <div
                  className="h-10 w-10 rounded-full border-4 border-indigo-100"
                  style={{
                    background: `conic-gradient(#4f46e5 ${countdownPercent}%, transparent 0)`,
                  }}
                />
              </div>
              <Button
                type="button"
                variant="outline"
                className="w-full h-11 rounded-xl font-medium"
                onClick={handleResendOTP}
                disabled={isResending || resendSecondsLeft > 0}
              >
                {isResending
                  ? "Resending..."
                  : resendSecondsLeft > 0
                  ? `Resend OTP in ${formattedCountdown}`
                  : "Resend OTP"}
              </Button>
            </div>
          </form>
        </div>
        </MagicCard>
      </div>
    </main>
  );
}

export default function PatientVerifyOTPPage() {
  return (
    <Suspense
      fallback={
        <main className="min-h-screen mesh-bg text-slate-950">
          <div className="flex flex-col items-center justify-center min-h-screen gap-6 px-4 py-8">
            <div className="glass-card p-8 rounded-3xl max-w-md w-full text-center">
              <h2 className="text-2xl font-black text-slate-950 mb-2">
                Verify Phone Number
              </h2>
              <p className="text-slate-500 text-sm">Loading security elements...</p>
            </div>
          </div>
        </main>
      }
    >
      <PatientVerifyOTPContent />
    </Suspense>
  );
}
