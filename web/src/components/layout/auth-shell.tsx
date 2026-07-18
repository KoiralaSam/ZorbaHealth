import Link from "next/link";
import type { LucideIcon } from "lucide-react";
import { ArrowLeft, BadgeCheck } from "lucide-react";
import { cn } from "../../lib/utils";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { AppHeader } from "./app-header";

type AuthShellProps = {
  eyebrow: string;
  title: string;
  description: string;
  featureTitle: string;
  featureItems: string[];
  featureIcon: LucideIcon;
  navLinks?: { href: string; label: string }[];
  step?: React.ReactNode;
  children: React.ReactNode;
  footer?: React.ReactNode;
};

export function AuthShell({
  eyebrow,
  title,
  description,
  featureTitle,
  featureItems,
  featureIcon: FeatureIcon,
  navLinks,
  step,
  children,
  footer,
}: AuthShellProps) {
  return (
    <main className="min-h-screen bg-slate-100 text-slate-950 dark:bg-slate-950 dark:text-slate-50">
      <div className="flex min-h-screen flex-col">
        <AppHeader
          links={navLinks}
          className="border-slate-200 bg-white/95 text-slate-950 shadow-sm dark:border-slate-800 dark:bg-slate-950/95 dark:text-white"
        />

        <div className="mx-auto grid w-full max-w-7xl flex-1 gap-8 px-5 py-8 lg:grid-cols-[0.8fr_1fr] lg:items-center">
          <aside className="hidden lg:block">
            <div className="rounded-2xl border border-slate-200 bg-white p-8 shadow-sm dark:border-slate-800 dark:bg-slate-950">
              <Badge className="mb-8 border-indigo-200 bg-indigo-50 text-indigo-700 dark:border-slate-700 dark:bg-slate-900 dark:text-indigo-200">
                {featureTitle}
              </Badge>
              <h1 className="text-4xl font-black tracking-tight text-slate-950 dark:text-white">{title}</h1>
              <p className="mt-4 max-w-xl text-sm leading-7 text-slate-600 dark:text-slate-400">
                {description}
              </p>
              <div className="mt-8 space-y-3">
                {featureItems.map((item, index) => (
                  <div
                    key={item}
                    className={cn(
                      "flex items-center gap-3 rounded-xl border border-slate-200 bg-slate-50 p-3 dark:border-slate-800 dark:bg-slate-900",
                      index === 0 && "border-indigo-200 bg-indigo-50 dark:border-indigo-900 dark:bg-indigo-950/30",
                    )}
                  >
                    <BadgeCheck className="h-4 w-4 text-emerald-600" />
                    <span className="text-sm font-semibold text-slate-700 dark:text-slate-200">
                      {item}
                    </span>
                  </div>
                ))}
              </div>
              <div className="mt-8 flex h-12 w-12 items-center justify-center rounded-xl bg-orange-50 ring-1 ring-orange-100 dark:bg-orange-950/30 dark:ring-orange-900">
                <FeatureIcon className="h-6 w-6 text-orange-600 dark:text-orange-300" />
              </div>
            </div>
          </aside>

          <section className="mx-auto w-full max-w-xl">
              <div className="rounded-2xl border border-slate-200 bg-white p-8 shadow-sm dark:border-slate-800 dark:bg-slate-950">
                <Button
                  asChild
                  variant="ghost"
                  className="mb-6 -ml-3 justify-start px-3 text-slate-500 hover:text-slate-900 dark:text-slate-300 dark:hover:text-white"
                >
                  <Link href="/">
                    <ArrowLeft className="h-4 w-4" />
                    Back to home
                  </Link>
                </Button>
                {step}
                <div className="mb-8">
                  <p className="text-xs font-black uppercase tracking-[0.26em] text-indigo-600 dark:text-indigo-300">
                    {eyebrow}
                  </p>
                  <h2 className="mt-2 text-3xl font-black tracking-tight text-slate-950 dark:text-slate-50">
                    {featureTitle}
                  </h2>
                  <p className="mt-2 text-sm leading-7 text-slate-500 dark:text-slate-400">
                    {description}
                  </p>
                </div>
                {children}
                {footer ? (
                  <div className="mt-8 border-t border-slate-200/70 pt-6 dark:border-slate-800">
                    {footer}
                  </div>
                ) : null}
              </div>
          </section>
        </div>
      </div>
    </main>
  );
}
