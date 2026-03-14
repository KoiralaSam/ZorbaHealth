"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "../../../components/ui/button";
import { API_URL } from "../../../constants";
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
        // date_of_birth is stored as RFC3339 string; convert back to YYYY-MM-DD if present
        dateOfBirth: parsed.date_of_birth
          ? parsed.date_of_birth.split("T")[0] ?? prev.dateOfBirth
          : prev.dateOfBirth,
        // we deliberately do NOT restore password fields for security reasons
        password: prev.password,
        confirmPassword: prev.confirmPassword,
      }));
    } catch {
      // If parsing fails, ignore and continue with empty form
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

    // Backend expects Go time.Time over JSON, which is an RFC3339 string.
    // The browser date input gives YYYY-MM-DD, so convert it to RFC3339.
    const dateOfBirthRFC3339 = formData.dateOfBirth
      ? `${formData.dateOfBirth}T00:00:00Z`
      : undefined;

    const patientRegisterRequest: HTTPPatientRegisterRequest = {
      phone_number: formData.phoneNumber,
      password: formData.password,
      full_name: formData.fullName,
      email: formData.email || undefined,
      date_of_birth: dateOfBirthRFC3339,
    };

    try {
      // patient registration API call
      const response = await fetch(
        `${API_URL}${APIEndpoints.PATIENT_REGISTER}`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(patientRegisterRequest),
        }
      );

      const data: HTTPPatientRegisterResponse = await response.json();

      if (response.ok) {
        if (typeof window !== "undefined") {
          window.sessionStorage.setItem(
            "patientRegistration",
            JSON.stringify(patientRegisterRequest)
          );
        }

        router.push(
          `/register/patient/otp?phone=${encodeURIComponent(formData.phoneNumber)}`
        );
      } else {
        setErrorMessage(
          data.error?.message ||
            "Registration failed. Please check your details and try again."
        );
      }
    } catch {
      setErrorMessage("Network error. Please try again.");
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <main className="min-h-screen bg-gradient-to-b from-blue-50 to-white">
      <div className="flex min-h-screen flex-col">
        <header className="w-full border-b border-blue-100 bg-white/70 backdrop-blur">
          <div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-4">
            <button
              type="button"
              onClick={() => router.push("/")}
              className="flex items-center gap-2"
            >
              <div className="flex h-9 w-9 items-center justify-center rounded-full bg-blue-100">
                <span className="text-base font-semibold text-blue-700">Z</span>
              </div>
              <span className="text-lg font-semibold text-gray-900">
                Zorba Health
              </span>
            </button>

            <nav className="flex items-center gap-3 text-sm">
              <button
                type="button"
                onClick={() => router.push("/login/patient")}
                className="text-gray-600 hover:text-gray-900"
              >
                Patient login
              </button>
              <button
                type="button"
                onClick={() => router.push("/login/hospital")}
                className="text-gray-600 hover:text-gray-900"
              >
                Hospital login
              </button>
            </nav>
          </div>
        </header>

        <div className="flex flex-1 items-center justify-center px-4 py-8">
          <div className="bg-white p-8 rounded-2xl shadow-lg max-w-md w-full">
            <div className="mb-6">
              <button
                onClick={() => router.push("/")}
                className="text-gray-500 hover:text-gray-700 mb-4 text-sm"
                type="button"
              >
                ← Back to home
              </button>
              <h2 className="text-2xl font-bold text-gray-900 mb-2">
                Patient Registration
              </h2>
              <p className="text-gray-600 text-sm">
                Create your account to access AI-powered health assistance.
              </p>
            </div>

            {errorMessage && (
              <div className="mb-4 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                {errorMessage}
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label
                htmlFor="fullName"
                className="block text-sm font-medium text-gray-700 mb-2"
              >
                Full Name *
              </label>
              <input
                id="fullName"
                type="text"
                value={formData.fullName}
                onChange={(e) =>
                  setFormData({ ...formData, fullName: e.target.value })
                }
                placeholder="John Doe"
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                required
              />
            </div>

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
                value={formData.phoneNumber}
                onChange={(e) =>
                  setFormData({ ...formData, phoneNumber: e.target.value })
                }
                placeholder="+1 (555) 123-4567"
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                required
              />
            </div>

            <div>
              <label
                htmlFor="email"
                className="block text-sm font-medium text-gray-700 mb-2"
              >
                Email Address (Optional)
              </label>
              <input
                id="email"
                type="email"
                value={formData.email}
                onChange={(e) =>
                  setFormData({ ...formData, email: e.target.value })
                }
                placeholder="john.doe@example.com"
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              />
            </div>

            <div>
              <label
                htmlFor="dateOfBirth"
                className="block text-sm font-medium text-gray-700 mb-2"
              >
                Date of Birth *
              </label>
              <input
                id="dateOfBirth"
                type="date"
                value={formData.dateOfBirth}
                onChange={(e) =>
                  setFormData({ ...formData, dateOfBirth: e.target.value })
                }
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                required
              />
            </div>

            <div>
              <label
                htmlFor="password"
                className="block text-sm font-medium text-gray-700 mb-2"
              >
                Password *
              </label>
              <input
                id="password"
                type="password"
                value={formData.password}
                onChange={(e) =>
                  setFormData({ ...formData, password: e.target.value })
                }
                placeholder="Create a strong password"
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                required
                minLength={8}
              />
            </div>

            <div>
              <label
                htmlFor="confirmPassword"
                className="block text-sm font-medium text-gray-700 mb-2"
              >
                Confirm Password *
              </label>
              <input
                id="confirmPassword"
                type="password"
                value={formData.confirmPassword}
                onChange={(e) =>
                  setFormData({
                    ...formData,
                    confirmPassword: e.target.value,
                  })
                }
                placeholder="Re-enter password"
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                required
                minLength={8}
              />
            </div>

              <Button
                type="submit"
                className="w-full text-lg py-6 bg-blue-600 hover:bg-blue-700 text-white"
                disabled={isLoading}
              >
                {isLoading ? "Creating Account..." : "Register"}
              </Button>
            </form>

            <div className="mt-6 text-center">
              <p className="text-sm text-gray-500">
                Already have an account?{" "}
                <button
                  onClick={() => router.push("/login/patient")}
                  className="text-blue-600 hover:text-blue-700 font-medium"
                  type="button"
                >
                  Login here
                </button>
              </p>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
