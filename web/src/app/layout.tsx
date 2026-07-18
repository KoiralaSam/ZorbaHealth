import type { Metadata, Viewport } from "next";
import { Inter } from "next/font/google";
import { ThemeProvider } from "../components/providers/theme-provider";
import { Toaster } from "../components/ui/sonner";
import "./globals.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
  display: "swap",
});

export const metadata: Metadata = {
  title: "Zorba Health | AI Voice Care Platform",
  description:
    "Secure AI-powered voice healthcare for patient record access, consent management, clinical summaries, and emergency monitoring.",
  applicationName: "Zorba Health",
  metadataBase: new URL("https://zorbahealth.app"),
  keywords: [
    "healthcare AI",
    "voice health assistant",
    "patient consent",
    "clinical summaries",
    "HIPAA",
  ],
  authors: [{ name: "Zorba Health" }],
  manifest: "/manifest.webmanifest",
  icons: {
    icon: [
      { url: "/favicon.svg", type: "image/svg+xml" },
      { url: "/brand/zorba-mark.png", sizes: "1024x1024", type: "image/png" },
    ],
    apple: [{ url: "/apple-touch-icon.png", sizes: "1024x1024", type: "image/png" }],
  },
};

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  themeColor: "#4F46E5",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="scroll-smooth" suppressHydrationWarning>
      <body className={`${inter.variable} antialiased`}>
        <ThemeProvider>
          <a
            href="#main-content"
            className="sr-only z-[100] rounded-full bg-slate-950 px-4 py-2 text-sm font-semibold text-white focus:not-sr-only focus:fixed focus:left-4 focus:top-4"
          >
            Skip to content
          </a>
          {children}
          <Toaster />
        </ThemeProvider>
      </body>
    </html>
  );
}
