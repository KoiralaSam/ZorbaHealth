"use client";

import { MoonStar, SunMedium } from "lucide-react";
import { useEffect, useState } from "react";
import { useTheme } from "next-themes";

export function ThemeToggle() {
  const { theme, resolvedTheme, setTheme } = useTheme();
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  const activeTheme = theme === "system" ? resolvedTheme : theme;

  if (!mounted) {
    return (
      <span
        aria-hidden="true"
        className="inline-flex h-10 w-10 rounded-full border border-border/70 bg-background/80 backdrop-blur"
      />
    );
  }

  return (
    <button
      type="button"
      aria-label={
        activeTheme === "dark" ? "Switch to light mode" : "Switch to dark mode"
      }
      className="inline-flex h-10 w-10 items-center justify-center rounded-full border border-border/70 bg-background/80 text-slate-700 shadow-sm backdrop-blur transition-colors hover:text-indigo-600 dark:text-slate-100 dark:hover:text-white"
      onClick={() => setTheme(activeTheme === "dark" ? "light" : "dark")}
    >
      {activeTheme === "dark" ? (
        <SunMedium className="h-4 w-4" />
      ) : (
        <MoonStar className="h-4 w-4" />
      )}
    </button>
  );
}
