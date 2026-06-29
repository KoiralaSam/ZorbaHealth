"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import {
  ArrowLeft,
  Calendar,
  HeartPulse,
  KeyRound,
  Mail,
  Phone,
  ShieldAlert,
  User,
} from "lucide-react";
import { API_URL } from "../../../constants";
import { DotPattern } from "../../../components/magicui/dot-pattern";
import { MagicCard } from "../../../components/magicui/magic-card";
import {
  APIEndpoints,
  HTTPPatientRegisterRequest,
  HTTPPatientRegisterResponse,
} from "../../../contracts";

export default function PatientRegister() {
  const router = useRouter();
  const [formData, setFormData] = useState({
    phoneNumber: "",
    email: "",
    fullName: "",
    dateOfBirth: "",
    password: "",
    confirmPassword: "",
  });
  const [isLoading, setIsLoading] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  useEffect(() => {
    if (typeof window === "undefined") return;

    try {
      const saved = window.sessionStorage.getItem("patientRegistration");
      if (!saved) return;
      const parsed = JSON.parse(saved) as HTTPPatientRegisterRequest;
      setFormData((prev) => ({
        ...prev,
        phoneNumber: parsed.phone_number || prev.phoneNumber,
        email: parsed.email || prev.email,
        fullName: parsed.full_name || prev.fullName,
        dateOfBirth: parsed.date_of_birth
          ? parsed.date_of_birth.split("T")[0] ?? prev.dateOfBirth
          : prev.dateOfBirth,
      }));
    } catch {
      // Ignore invalid saved registration data.
    }
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMessage(null);

    if (formData.password !== formData.confirmPassword) {
      setErrorMessage("Passwords do not match.");
      return;
    }

    setIsLoading(true);

    const patientRegisterRequest: HTTPPatientRegisterRequest = {
      phone_number: formData.phoneNumber,
      password: formData.password,
      full_name: formData.fullName,
      email: formData.email || undefined,
      date_of_birth: formData.dateOfBirth
        ? `${formData.dateOfBirth}T00:00:00Z`
        : undefined,
    };

    try {
      const response = await fetch(
        `${API_URL}${APIEndpoints.PATIENT_REGISTER}`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(patientRegisterRequest),
        },
      );

      const data: HTTPPatientRegisterResponse = await response.json();

      if (response.ok) {
        window.sessionStorage.setItem(
          "patientRegistration",
          JSON.stringify(patientRegisterRequest),
        );
        router.push(
          `/register/patient/otp?phone=${encodeURIComponent(formData.phoneNumber)}`,
        );
      } else {
        setErrorMessage(
          data.error?.message ||
            "Registration failed. Please check your details and try again.",
        );
      }
    } catch {
      setErrorMessage("Network error. Please try again.");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <main className="relative min-h-screen overflow-hidden bg-slate-950 text-slate-950">
      <div className="absolute inset-0 bg-[radial-gradient(circle_at_top_left,rgba(99,102,241,0.28),transparent_24%),radial-gradient(circle_at_top_right,rgba(249,115,22,0.18),transparent_18%),linear-gradient(180deg,#020617_0%,#0f172a_55%,#111827_100%)]" />
      <DotPattern
        glow
        className="text-indigo-300/30 [mask-image:radial-gradient(700px_circle_at_center,white,transparent)]"
      />
      <header className="relative border-b border-white/10 bg-slate-950/70 backdrop-blur-xl">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-5 py-4">
          <button
            type="button"
            onClick={() => router.push("/")}
            className="flex items-center gap-3"
          >
            <div className="flex h-10 w-10 items-center justify-center rounded-2xl bg-gradient-to-br from-indigo-600 to-orange-500 text-white shadow-glow">
              <HeartPulse className="h-5 w-5" />
            </div>
            <span className="text-lg font-black tracking-tight gradient-text">
              Zorba Health
            </span>
          </button>
          <button
            type="button"
            onClick={() => router.push("/login/patient")}
            className="text-sm font-bold text-slate-600 hover:text-indigo-600"
          >
            Sign In
          </button>
        </div>
      </header>

      <div className="relative mx-auto flex min-h-[calc(100vh-73px)] max-w-3xl items-center px-5 py-8">
        <MagicCard
          className="w-full overflow-hidden rounded-3xl p-0"
          gradientFrom="#4f46e5"
          gradientTo="#f97316"
          gradientColor="rgba(79, 70, 229, 0.12)"
        >
          <section className="relative w-full rounded-3xl border border-white/60 bg-white/90 p-6 shadow-2xl shadow-slate-950/20 backdrop-blur-xl sm:p-8 dark:border-slate-800 dark:bg-slate-950/90">
          <button
            onClick={() => router.push("/")}
            className="mb-6 inline-flex items-center gap-1.5 text-sm font-bold text-slate-500 hover:text-slate-900"
            type="button"
          >
            <ArrowLeft className="h-4 w-4" /> Back to home
          </button>

          <div className="mb-8">
            <p className="text-xs font-black uppercase tracking-wide text-indigo-600">
              Step 1 of 3
            </p>
            <h1 className="mt-2 text-3xl font-black tracking-tight">
              Create your patient account
            </h1>
            <div className="mt-5 grid grid-cols-3 gap-2">
              {["Info", "Verify Phone", "Verify Email"].map((step, index) => (
                <div key={step}>
                  <div
                    className={`h-2 rounded-full ${
                      index === 0 ? "bg-indigo-600" : "bg-slate-200"
                    }`}
                  />
                  <p className="mt-2 text-xs font-bold text-slate-500">{step}</p>
                </div>
              ))}
            </div>
          </div>

          {errorMessage ? (
            <div className="mb-6 flex items-start gap-2 rounded-xl border border-rose-200 bg-rose-50/70 p-4 text-sm text-rose-700">
              <ShieldAlert className="mt-0.5 h-5 w-5 shrink-0 text-rose-600" />
              <span>{errorMessage}</span>
            </div>
          ) : null}

          <form onSubmit={handleSubmit} className="space-y-5">
            <div className="grid gap-5 sm:grid-cols-2">
              <Input
                label="Full Name"
                icon={<User className="h-5 w-5" />}
                value={formData.fullName}
                onChange={(e) =>
                  setFormData({ ...formData, fullName: e.target.value })
                }
                placeholder="John Doe"
                required
              />
              <Input
                label="Phone Number"
                icon={<Phone className="h-5 w-5" />}
                type="tel"
                value={formData.phoneNumber}
                onChange={(e) =>
                  setFormData({ ...formData, phoneNumber: e.target.value })
                }
                placeholder="+15551234567"
                required
              />
              <Input
                label="Email Address"
                icon={<Mail className="h-5 w-5" />}
                type="email"
                value={formData.email}
                onChange={(e) =>
                  setFormData({ ...formData, email: e.target.value })
                }
                placeholder="john.doe@example.com"
              />
              <Input
                label="Date of Birth"
                icon={<Calendar className="h-5 w-5" />}
                type="date"
                value={formData.dateOfBirth}
                onChange={(e) =>
                  setFormData({ ...formData, dateOfBirth: e.target.value })
                }
                required
              />
              <Input
                label="Password"
                icon={<KeyRound className="h-5 w-5" />}
                type="password"
                value={formData.password}
                onChange={(e) =>
                  setFormData({ ...formData, password: e.target.value })
                }
                placeholder="Minimum 8 characters"
                minLength={8}
                required
              />
              <Input
                label="Confirm Password"
                icon={<KeyRound className="h-5 w-5" />}
                type="password"
                value={formData.confirmPassword}
                onChange={(e) =>
                  setFormData({ ...formData, confirmPassword: e.target.value })
                }
                placeholder="Repeat password"
                minLength={8}
                required
              />
            </div>

            <Button
              type="submit"
              variant="healthcare"
              className="h-12 w-full text-base"
              disabled={isLoading}
            >
              {isLoading ? "Creating Account..." : "Continue to Phone Verification"}
            </Button>
          </form>
        </section>
        </MagicCard>
      </div>
    </main>
  );
}
