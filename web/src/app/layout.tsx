import type { Metadata, Viewport } from "next";
import { ThemeProvider } from "../components/providers/theme-provider";
import { Toaster } from "../components/ui/sonner";
import "./globals.css";

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
  themeColor: "#4338CA",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="scroll-smooth" suppressHydrationWarning>
      <body className={`antialiased`}>
        <ThemeProvider>
          <a
            href="#main-content"
            className="skip-link"
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
