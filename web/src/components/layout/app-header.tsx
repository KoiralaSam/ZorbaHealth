"use client";

import Link from "next/link";
import { ThemeToggle } from "../theme-toggle";
import { Brand } from "./brand";
import { cn } from "../../lib/utils";

type NavLink = {
  href: string;
  label: string;
  className?: string;
};

type AppHeaderProps = {
  links?: NavLink[];
  rightSlot?: React.ReactNode;
  className?: string;
};

export function AppHeader({ links = [], rightSlot, className }: AppHeaderProps) {
  return (
    <header
      className={cn(
        "sticky top-0 z-40 w-full border-b border-white/60 bg-background/80 backdrop-blur-xl dark:border-slate-800/70 dark:bg-slate-950/80",
        className,
      )}
    >
      <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-5 py-4">
        <Brand />
        <div className="flex items-center gap-3">
          {links.length ? (
            <nav className="hidden items-center gap-5 text-sm font-bold text-slate-600 dark:text-slate-300 sm:flex">
              {links.map((link) => (
                <Link
                  key={link.href + link.label}
                  href={link.href}
                  className={cn(
                    "transition-colors hover:text-indigo-600 dark:hover:text-indigo-300",
                    link.className,
                  )}
                >
                  {link.label}
                </Link>
              ))}
            </nav>
          ) : null}
          {rightSlot}
          <ThemeToggle />
        </div>
      </div>
    </header>
  );
}
