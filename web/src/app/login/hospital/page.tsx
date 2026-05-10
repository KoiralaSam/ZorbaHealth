"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Button } from "../../../components/ui/button";
import { HTTPHospitalLoginRequest, APIEndpoints } from "../../../contracts";
import { API_URL } from "../../../constants";

export default function HospitalLogin() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);

    const hospitalLoginRequest: HTTPHospitalLoginRequest = {
      email,
      password,
    };

    try {
      const response = await fetch(`${API_URL}${APIEndpoints.HOSPITAL_LOGIN}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify(hospitalLoginRequest),
      });

      const data: { error?: { message?: string } } = await response.json();

      if (response.ok) {
        // router.push("/hospital/dashboard");
        alert("Hospital login successful!");
      } else {
        alert(`Login failed: ${data.error?.message || "Unknown error"}`);
      }
    } catch {
      alert("Network error - Please try again");
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
                onClick={() => router.push("/register/hospital")}
                className="text-gray-600 hover:text-gray-900"
              >
                Hospital sign up
              </button>
            </nav>
          </div>
        </header>

        <div className="flex flex-1 items-center justify-center px-4 py-8">
          <div className="bg-white p-8 rounded-2xl shadow-lg max-w-md w-full">
            <div className="mb-6">
              <button
                onClick={() => router.push("/")}
                className="mb-4 text-sm text-gray-500 hover:text-gray-700"
              >
                ← Back to home
              </button>
              <h2 className="text-2xl font-bold text-gray-900 mb-2">
                Hospital Login
              </h2>
              <p className="text-gray-600 text-sm">
                Access the health service portal to manage patient records and
                analytics
              </p>
            </div>

            <form onSubmit={handleLogin} className="space-y-4">
              <div>
                <label
                  htmlFor="email"
                  className="block text-sm font-medium text-gray-700 mb-2"
                >
                  Email Address
                </label>
                <input
                  id="email"
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  placeholder="staff@hospital.com"
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

              <div className="flex items-center justify-between text-sm">
                <label className="flex items-center">
                  <input type="checkbox" className="mr-2" />
                  <span className="text-gray-600">Remember me</span>
                </label>
                <button
                  type="button"
                  className="text-blue-600 hover:text-blue-700"
                >
                  Forgot password?
                </button>
              </div>

              <Button
                type="submit"
                className="w-full text-lg py-6 bg-blue-600 hover:bg-blue-700 text-white"
                disabled={isLoading}
              >
                {isLoading ? "Logging in..." : "Sign In"}
              </Button>
            </form>

            <div className="mt-6 text-center space-y-2">
              <p className="text-sm text-gray-500">
                New hospital?{" "}
                <button
                  onClick={() => router.push("/register/hospital")}
                  className="text-blue-600 hover:text-blue-700 font-medium"
                  type="button"
                >
                  Register here
                </button>
              </p>
              <p className="text-sm text-gray-500">
                Need help?{" "}
                <button
                  className="text-blue-600 hover:text-blue-700 font-medium"
                  type="button"
                >
                  Contact support
                </button>
              </p>
            </div>
          </div>
        </div>
      </div>
    </main>
  );
}
