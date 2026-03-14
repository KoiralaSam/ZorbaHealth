"use client";

import { Suspense, useEffect, useMemo, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Button } from "../../../../components/ui/button";
import { API_URL } from "../../../../constants";
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
  const [resendSecondsLeft, setResendSecondsLeft] = useState(120);
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
        setResendSecondsLeft(120);
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

  return (
    <main className="min-h-screen bg-gradient-to-b from-blue-50 to-white">
      <div className="flex flex-col items-center justify-center min-h-screen gap-6 px-4 py-8">
        <div className="bg-white p-8 rounded-2xl shadow-lg max-w-md w-full">
          <button
            onClick={() => router.push("/register/patient")}
            className="text-gray-500 hover:text-gray-700 mb-4"
          >
            ← Back
          </button>

          <h2 className="text-2xl font-bold text-gray-900 mb-2">
            Verify your phone
          </h2>
          <p className="text-gray-600 text-sm mb-6">
            Enter the OTP we sent to your phone number to continue. You can
            update your phone number below and request a new OTP if needed.
          </p>

          {statusMessage && (
            <div
              className={`mb-4 rounded-lg border px-4 py-3 text-sm ${
                statusType === "error"
                  ? "border-red-200 bg-red-50 text-red-700"
                  : statusType === "success"
                  ? "border-green-200 bg-green-50 text-green-700"
                  : "border-blue-200 bg-blue-50 text-blue-700"
              }`}
            >
              {statusMessage}
            </div>
          )}

          <form onSubmit={handleVerify} className="space-y-4">
            <div>
              <label
                htmlFor="phone"
                className="block text-sm font-medium text-gray-700 mb-2"
              >
                Phone Number *
              </label>
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
                placeholder="+1 (555) 123-4567"
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                required
              />
            </div>

            <div>
              <label
                htmlFor="otp"
                className="block text-sm font-medium text-gray-700 mb-2"
              >
                OTP *
              </label>
              <input
                id="otp"
                inputMode="numeric"
                value={otp}
                onChange={(e) => setOtp(e.target.value)}
                placeholder="6-digit code"
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent tracking-widest text-center text-lg"
                required
              />
            </div>

            <Button
              type="submit"
              className="w-full text-lg py-6 bg-blue-600 hover:bg-blue-700 text-white"
              disabled={isLoading}
            >
              {isLoading ? "Verifying..." : "Verify OTP"}
            </Button>

            <div className="pt-2 border-t border-gray-100">
              <p className="text-xs text-gray-500 mb-2">
                Didn&apos;t receive a code? You can request a new OTP every 2
                minutes.
              </p>
              <div className="flex items-center justify-between gap-3">
                <Button
                  type="button"
                  variant="outline"
                  className="flex-1"
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
            </div>
          </form>
        </div>
      </div>
    </main>
  );
}

export default function PatientVerifyOTPPage() {
  return (
    <Suspense
      fallback={
        <main className="min-h-screen bg-gradient-to-b from-blue-50 to-white">
          <div className="flex flex-col items-center justify-center min-h-screen gap-6 px-4 py-8">
            <div className="bg-white p-8 rounded-2xl shadow-lg max-w-md w-full text-center">
              <h2 className="text-2xl font-bold text-gray-900 mb-2">
                Verify your phone
              </h2>
              <p className="text-gray-600 text-sm">Loading…</p>
            </div>
          </div>
        </main>
      }
    >
      <PatientVerifyOTPContent />
    </Suspense>
  );
}

