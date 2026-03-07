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
      <div className="flex flex-col items-center justify-center h-screen gap-6 px-4">
        <div className="bg-white p-8 rounded-2xl shadow-lg text-center max-w-md w-full">
          <div className="mb-6">
            <div className="w-16 h-16 bg-blue-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <svg
                className="w-8 h-8 text-blue-600"
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
            <h1 className="text-3xl font-bold text-gray-900 mb-2">
              Zorba Health
            </h1>
            <p className="text-gray-600">AI-Powered Voice Health Assistant</p>
          </div>

          <div className="space-y-4 mt-8">
            <Button
              className="w-full text-lg py-6 bg-blue-600 hover:bg-blue-700 text-white"
              onClick={() => handleLoginType("patient")}
            >
              Patient Login
            </Button>
            <Button
              className="w-full text-lg py-6"
              variant="outline"
              onClick={() => handleLoginType("hospital")}
            >
              Hospital Login
            </Button>
          </div>
        </div>
      </div>
    </main>
  );
}
