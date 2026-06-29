"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { AuthShell } from "../../../components/layout/auth-shell";
import { StatusBanner } from "../../../components/status-banner";
import {
  KeyRound,
  Mail,
  Stethoscope,
} from "lucide-react";
import {
  APIEndpoints,
  HTTPHospitalLoginRequest,
  HTTPHospitalLoginResponse,
} from "../../../contracts";
import { API_URL } from "../../../constants";
import { setAuth } from "../../../lib/auth-client";

const schema = z.object({
  email: z.email("Enter a valid clinical email address."),
  password: z.string().min(8, "Password must be at least 8 characters."),
});

type FormValues = z.infer<typeof schema>;

export default function HospitalLogin() {
  const router = useRouter();
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      email: "",
      password: "",
    },
  });

  const handleLogin = async ({ email, password }: FormValues) => {
    const hospitalLoginRequest: HTTPHospitalLoginRequest = {
      email,
      password,
    };

    try {
      const response = await fetch(`${API_URL}${APIEndpoints.HOSPITAL_LOGIN}`, {
        method: "POST",
        credentials: "include",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(hospitalLoginRequest),
      });

      const data: HTTPHospitalLoginResponse = await response.json();

      if (response.ok) {
        const accessToken = data.data?.access_token;
        if (!accessToken) {
          setError("root", {
            message: "Login succeeded but no access token was returned.",
          });
          return;
        }
        setAuth({
          role: "hospital_staff",
          accessToken,
          hospitalId: data.data?.hospital_id,
          staffId: data.data?.staff_id,
          staffRole: data.data?.role,
        });
        router.replace("/hospital/dashboard");
      } else {
        setError("root", {
          message: data.error?.message || "Invalid email or password.",
        });
      }
    } catch {
      setError("root", {
        message: "Network error. Please try again.",
      });
    }
  };

  return (
    <AuthShell
      eyebrow="Clinical access"
      title="Clinical summaries and incident monitoring in one workspace."
      description="Log in as hospital staff to inspect patient health record summaries, audit trail logs, and emergency incidents."
      featureTitle="Hospital Login"
      featureItems={["Patient summary generation", "Emergency incident queue", "Audit trail search"]}
      featureIcon={Stethoscope}
      navLinks={[
        { href: "/login/patient", label: "Patient Login" },
        { href: "/register/hospital", label: "Register Hospital" },
      ]}
      footer={
        <p className="text-center text-sm text-slate-500 dark:text-slate-400">
          New hospital?{" "}
          <Link href="/register/hospital" className="font-bold text-indigo-600 hover:text-indigo-700">
            Register here
          </Link>
        </p>
      }
    >
      {errors.root?.message ? <StatusBanner tone="error" message={errors.root.message} className="mb-6" /> : null}

      <form onSubmit={handleSubmit(handleLogin)} className="space-y-5">
        <Input
          label="Clinical Email Address"
          icon={<Mail className="h-5 w-5" />}
          type="email"
          placeholder="staff@hospital.com"
          autoComplete="username"
          error={errors.email?.message}
          {...register("email")}
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
          {isSubmitting ? "Signing in..." : "Sign In"}
        </Button>
      </form>
    </AuthShell>
  );
}
