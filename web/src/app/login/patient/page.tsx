"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { StatusBanner } from "../../../components/status-banner";
import { AuthShell } from "../../../components/layout/auth-shell";
import {
  KeyRound,
  Mail,
  ShieldCheck,
  UserRound,
} from "lucide-react";
import {
  HTTPPatientLoginRequest,
  HTTPPatientLoginResponse,
  APIEndpoints,
} from "../../../contracts";
import { API_URL } from "../../../constants";
import { setAuth } from "../../../lib/auth-client";

const schema = z.object({
  identifier: z.string().min(3, "Enter your email address or phone number."),
  password: z.string().min(8, "Password must be at least 8 characters."),
});

type FormValues = z.infer<typeof schema>;

export default function PatientLogin() {
  const router = useRouter();
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      identifier: "",
      password: "",
    },
  });

  const handleLogin = async ({ identifier, password }: FormValues) => {
    const patientLoginRequest: HTTPPatientLoginRequest = {
      identifier: identifier.trim(),
      password,
    };

    try {
      const response = await fetch(`${API_URL}${APIEndpoints.PATIENT_LOGIN}`, {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(patientLoginRequest),
      });

      const data: HTTPPatientLoginResponse = await response.json();
      if (!response.ok) {
        setError("root", {
          message: data.error?.message || "Login failed. Please try again.",
        });
        return;
      }

      const accessToken = data.data?.access_token;
      const patientID = data.data?.patient_id;
      if (!accessToken || !patientID) {
        setError("root", {
          message: "Login succeeded but no patient session was returned.",
        });
        return;
      }

      setAuth({ role: "patient", accessToken, patientId: patientID });
      router.replace("/patient");
    } catch (error) {
      console.error("Network error - Please try again", error);
      setError("root", {
        message: "Network error. Please try again.",
      });
    }
  };

  return (
    <AuthShell
      eyebrow="Patient sign in"
      title="Secure access to your voice care workspace."
      description="Log in to securely consult your record summaries, manage consents, and review your voice assistant activity."
      featureTitle="Patient Portal"
      featureItems={["Consent controls", "Record Q&A", "Call history", "GPS safety sessions"]}
      featureIcon={ShieldCheck}
      navLinks={[
        { href: "/register/patient", label: "Sign Up" },
        { href: "/login/hospital", label: "Hospital Portal" },
      ]}
      footer={
        <p className="text-center text-sm text-slate-500 dark:text-slate-400">
          New patient?{" "}
          <Link href="/register/patient" className="font-bold text-indigo-600 hover:text-indigo-700">
            Register here
          </Link>
        </p>
      }
    >
      {errors.root?.message ? <StatusBanner tone="error" message={errors.root.message} className="mb-6" /> : null}

      <form onSubmit={handleSubmit(handleLogin)} className="space-y-5">
        <Input
          label="Email or Phone Number"
          icon={<Mail className="h-5 w-5" />}
          placeholder="you@example.com or +15551234567"
          autoComplete="username"
          error={errors.identifier?.message}
          {...register("identifier")}
        />

        <Input
          label="Password"
          icon={<KeyRound className="h-5 w-5" />}
          type="password"
          placeholder="Enter your account password"
          autoComplete="current-password"
          error={errors.password?.message}
          {...register("password")}
        />

        <Button type="submit" variant="healthcare" className="h-12 w-full text-base" disabled={isSubmitting}>
          <UserRound className="h-4 w-4" />
          {isSubmitting ? "Signing in..." : "Continue"}
        </Button>
      </form>
    </AuthShell>
  );
}
