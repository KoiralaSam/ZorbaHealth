"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import {
  ArrowLeft,
  Building2,
  FileText,
  KeyRound,
  Mail,
  MapPin,
  Phone,
  ShieldCheck,
} from "lucide-react";
import { DotPattern } from "../../../components/magicui/dot-pattern";
import { MagicCard } from "../../../components/magicui/magic-card";
import {
  APIEndpoints,
  HTTPHospitalRegisterRequest,
  HTTPHospitalRegisterResponse,
} from "../../../contracts";
import { API_URL } from "../../../constants";

export default function HospitalRegister() {
  const router = useRouter();
  const [formData, setFormData] = useState({
    hospitalName: "",
    staffName: "",
    email: "",
    password: "",
    confirmPassword: "",
    contactNumber: "",
    address: "",
    registrationNumber: "",
  });
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setNotice("");

    if (formData.password !== formData.confirmPassword) {
      setError("Passwords do not match.");
      return;
    }

    setIsLoading(true);
    const payload: HTTPHospitalRegisterRequest = {
      hospital_name: formData.hospitalName.trim(),
      license_no: formData.registrationNumber.trim(),
      email: formData.email.trim(),
      phone_number: formData.contactNumber.trim(),
      password: formData.password,
      staff_name: formData.staffName.trim(),
      staff_role: "admin",
      address: formData.address.trim(),
    };

    try {
      const response = await fetch(`${API_URL}${APIEndpoints.HOSPITAL_REGISTER}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });
      const data: HTTPHospitalRegisterResponse = await response.json();
      if (!response.ok) {
        setError(data.error?.message || "Hospital registration failed.");
        return;
      }
      setNotice("Hospital registered. Your admin staff login is ready.");
      router.push("/login/hospital");
    } catch {
      setError("Network error. Please try again.");
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
      <div className="relative mx-auto flex min-h-screen max-w-4xl items-center px-5 py-8">
        <MagicCard
          className="w-full overflow-hidden rounded-3xl p-0"
          gradientFrom="#4f46e5"
          gradientTo="#f97316"
          gradientColor="rgba(79, 70, 229, 0.12)"
        >
        <section className="relative w-full rounded-3xl border border-white/60 bg-white/90 p-6 shadow-2xl shadow-slate-950/20 backdrop-blur-xl sm:p-8 dark:border-slate-800 dark:bg-slate-950/90">
          <div className="mb-8 flex flex-col gap-5 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <button
                onClick={() => router.push("/")}
                className="mb-6 inline-flex items-center gap-1.5 text-sm font-bold text-slate-500 hover:text-slate-900"
                type="button"
              >
                <ArrowLeft className="h-4 w-4" /> Back to home
              </button>
              <div className="mb-4 flex h-14 w-14 items-center justify-center rounded-2xl bg-slate-950 text-white">
                <Building2 className="h-6 w-6" />
              </div>
              <p className="text-xs font-black uppercase tracking-wide text-indigo-600">
                Healthcare partner onboarding
              </p>
              <h1 className="mt-2 text-3xl font-black tracking-tight">
                Hospital Registration
              </h1>
              <p className="mt-2 max-w-2xl text-sm leading-7 text-slate-600">
                Submit your facility details for compliance review and access to
                clinical summaries, audit search, and incident workflows.
              </p>
            </div>
            <button
              type="button"
              onClick={() => router.push("/login/hospital")}
              className="text-sm font-bold text-indigo-600 hover:text-indigo-700"
            >
              Hospital Login
            </button>
          </div>

          {error ? (
            <div className="mb-6 rounded-xl border border-rose-200 bg-rose-50/70 p-4 text-sm font-semibold text-rose-700">
              {error}
            </div>
          ) : null}
          {notice ? (
            <div className="mb-6 rounded-xl border border-emerald-200 bg-emerald-50/70 p-4 text-sm font-semibold text-emerald-700">
              {notice}
            </div>
          ) : null}

          <form onSubmit={handleSubmit} className="space-y-5">
            <div className="grid gap-5 md:grid-cols-2">
              <Input
                label="Hospital Name"
                icon={<Building2 className="h-5 w-5" />}
                value={formData.hospitalName}
                onChange={(e) =>
                  setFormData({ ...formData, hospitalName: e.target.value })
                }
                placeholder="General Hospital"
                required
              />
              <Input
                label="Registration Number"
                icon={<FileText className="h-5 w-5" />}
                value={formData.registrationNumber}
                onChange={(e) =>
                  setFormData({
                    ...formData,
                    registrationNumber: e.target.value,
                  })
                }
                placeholder="REG123456"
                required
              />
              <Input
                label="Official Email"
                icon={<Mail className="h-5 w-5" />}
                type="email"
                value={formData.email}
                onChange={(e) =>
                  setFormData({ ...formData, email: e.target.value })
                }
                placeholder="admin@hospital.com"
                required
              />
              <Input
                label="Admin Staff Name"
                icon={<ShieldCheck className="h-5 w-5" />}
                value={formData.staffName}
                onChange={(e) =>
                  setFormData({ ...formData, staffName: e.target.value })
                }
                placeholder="Dr. Jane Admin"
                required
              />
              <Input
                label="Contact Number"
                icon={<Phone className="h-5 w-5" />}
                type="tel"
                value={formData.contactNumber}
                onChange={(e) =>
                  setFormData({ ...formData, contactNumber: e.target.value })
                }
                placeholder="+15550000000"
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
                placeholder="Create password"
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

            <div className="space-y-2">
              <label
                htmlFor="address"
                className="text-xs font-bold uppercase tracking-wide text-slate-600"
              >
                Address
              </label>
              <div className="relative">
                <MapPin className="absolute left-3 top-3.5 h-5 w-5 text-slate-400" />
                <textarea
                  id="address"
                  value={formData.address}
                  onChange={(e) =>
                    setFormData({ ...formData, address: e.target.value })
                  }
                  placeholder="123 Medical Center Dr, City, State, ZIP"
                  className="min-h-24 w-full rounded-xl border border-slate-200 bg-white px-4 py-3 pl-10 text-sm outline-none transition-all placeholder:text-slate-400 focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100"
                  required
                />
              </div>
            </div>

            <div className="rounded-2xl border border-indigo-100 bg-indigo-50/70 p-4">
              <div className="flex items-start gap-3">
                <ShieldCheck className="mt-0.5 h-5 w-5 shrink-0 text-indigo-600" />
                <p className="text-sm font-semibold leading-6 text-indigo-950">
                  Healthcare partner registrations are subject to compliance
                  verification before staff access is enabled.
                </p>
              </div>
            </div>

            <Button
              type="submit"
              variant="healthcare"
              className="h-12 w-full text-base"
              disabled={isLoading}
            >
              {isLoading ? "Submitting..." : "Submit Registration"}
            </Button>
          </form>
        </section>
        </MagicCard>
      </div>
    </main>
  );
}
