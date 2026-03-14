"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "../../../components/ui/button";
import {
  HTTPPatientLoginRequest,
  HTTPPatientLoginResponse,
  APIEndpoints,
} from "../../../contracts";
import { API_URL } from "../../../constants";

export default function PatientLogin() {
  const router = useRouter();
  const [phoneNumber, setPhoneNumber] = useState("");
  const [password, setPassword] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    const patientLoginRequest: HTTPPatientLoginRequest = {
      phone_number: phoneNumber,
      password,
    };

    try {
      const response = await fetch(`${API_URL}${APIEndpoints.PATIENT_LOGIN}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(patientLoginRequest),
      });

      const data: HTTPPatientLoginResponse = await response.json();
      console.log(data);
    } catch (error) {
      console.error("Network error - Please try again", error);
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
                onClick={() => router.push("/register/patient")}
                className="text-gray-600 hover:text-gray-900"
              >
                Patient sign up
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
                Patient Login
              </h2>
              <p className="text-gray-600 text-sm">
                Enter your phone number and password to access your health
                records and voice assistant.
              </p>
            </div>

            <form onSubmit={handleLogin} className="space-y-4">
            <div>
              <label
                htmlFor="phone"
                className="block text-sm font-medium text-gray-700 mb-2"
              >
                Phone Number
              </label>
              <input
                id="phone"
                type="tel"
                value={phoneNumber}
                onChange={(e) => setPhoneNumber(e.target.value)}
                placeholder="+1 (555) 123-4567"
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                required
              />
            </div>

            <div>
              <label
                htmlFor="password"
                className="block text-sm font-medium text-gray-700 mb-2"
              >
                Password
              </label>
              <input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Enter your password"
                className="w-full px-4 py-3 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                required
              />
            </div>

              <Button
                type="submit"
                className="w-full text-lg py-6 bg-blue-600 hover:bg-blue-700 text-white"
                disabled={isLoading}
              >
                {isLoading ? "Logging in..." : "Continue"}
              </Button>
            </form>

            <div className="mt-6 text-center">
              <p className="text-sm text-gray-500">
                New patient?{" "}
                <button
                  onClick={() => router.push("/register/patient")}
                  className="text-blue-600 hover:text-blue-700 font-medium"
                  type="button"
                >
                  Register here
                </button>
              </p>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
