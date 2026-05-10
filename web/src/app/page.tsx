"use client";

import { useRouter } from "next/navigation";
import { Button } from "../components/ui/button";

export default function Home() {
  const router = useRouter();

  const handleLoginType = (type: "patient" | "hospital") => {
    if (type === "patient") {
      router.push("/login/patient");
    } else {
      router.push("/login/hospital");
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
                <svg
                  className="h-5 w-5 text-blue-600"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth="2"
                    d="M19 21V5a2 2 0 00-2-2H7a2 2 0 00-2 2v16m14 0h2m-2 0h-5m-9 0H3m2 0h5M9 7h1m-1 4h1m4-4h1m-1 4h1m-5 10v-5a1 1 0 011-1h2a1 1 0 011 1v5m-4 0h4"
                  />
                </svg>
              </div>
              <span className="text-lg font-semibold text-gray-900">
                Zorba Health
              </span>
            </button>

            <nav className="flex items-center gap-3 text-sm">
              <button
                type="button"
                onClick={() => handleLoginType("patient")}
                className="text-gray-600 hover:text-gray-900"
              >
                Patient login
              </button>
              <button
                type="button"
                onClick={() => handleLoginType("hospital")}
                className="text-gray-600 hover:text-gray-900"
              >
                Hospital login
              </button>
            </nav>
          </div>
        </header>

        <div className="flex flex-1 items-center justify-center px-4 py-8">
          <div className="mx-auto grid w-full max-w-5xl gap-10 md:grid-cols-[minmax(0,1.3fr)_minmax(0,1fr)]">
            <section className="flex flex-col justify-center">
              <p className="mb-3 text-sm font-medium uppercase tracking-wide text-blue-600">
                AI-powered voice health assistant
              </p>
              <h1 className="mb-4 text-3xl font-bold text-gray-900 md:text-4xl">
                Better health conversations,
                <br className="hidden md:inline" /> right from your phone.
              </h1>
              <p className="mb-6 text-sm text-gray-600 md:text-base">
                Zorba Health helps patients and hospitals manage appointments,
                reminders, and follow-ups using natural voice interactions. No
                apps to install, just simple and secure communication.
              </p>
              <div className="flex flex-col gap-3 sm:flex-row">
                <Button
                  className="w-full py-6 text-base sm:w-auto"
                  onClick={() => router.push("/register/patient")}
                >
                  Patient sign up
                </Button>
                <Button
                  variant="outline"
                  className="w-full py-6 text-base sm:w-auto"
                  onClick={() => router.push("/register/hospital")}
                >
                  Hospital sign up
                </Button>
              </div>
            </section>

            <section className="flex items-center justify-center">
              <div className="w-full max-w-sm rounded-2xl bg-white p-5 shadow-md">
                <h2 className="mb-1 text-base font-semibold text-gray-900">
                  Quick access
                </h2>
                <p className="mb-4 text-xs text-gray-600">
                  Choose how you want to continue.
                </p>
                <div className="space-y-2">
                  <Button
                    className="w-full py-3 text-sm"
                    onClick={() => handleLoginType("patient")}
                  >
                    Patient login
                  </Button>
                  <Button
                    variant="outline"
                    className="w-full py-3 text-sm"
                    onClick={() => handleLoginType("hospital")}
                  >
                    Hospital login
                  </Button>
                </div>
              </div>
            </section>
          </div>
        </div>
      </div>
    </main>
  );
}
