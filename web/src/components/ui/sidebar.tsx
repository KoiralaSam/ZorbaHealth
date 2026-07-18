"use client";

import Link from "next/link";
import type { LucideIcon } from "lucide-react";
import { Menu, X } from "lucide-react";
import { useState } from "react";
import { cn } from "../../lib/utils";

export type SidebarItem = {
  id: string;
  label: string;
  icon: LucideIcon;
  href?: string;
};

type SidebarProps = {
  title: string;
  subtitle: string;
  items: SidebarItem[];
  activeItem: string;
  onChange?: (item: string) => void;
};

export function Sidebar({
  title,
  subtitle,
  items,
  activeItem,
  onChange,
}: SidebarProps) {
  const [open, setOpen] = useState(false);

  const nav = (
    <nav className="space-y-2">
      {items.map((item) => {
        const Icon = item.icon;
        const active = activeItem === item.id;
        return item.href ? (
            <Link
              key={item.id}
              href={item.href}
              onClick={() => {
                onChange?.(item.id);
                setOpen(false);
              }}
              className={cn(
                "flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left text-sm font-bold transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
                active
                  ? "bg-indigo-600 text-white shadow-glow"
                  : "text-slate-600 hover:bg-indigo-50 hover:text-indigo-700 dark:text-slate-300 dark:hover:bg-slate-900 dark:hover:text-white",
              )}
            >
              <Icon className="h-4 w-4" />
              <span>{item.label}</span>
            </Link>
          ) : (
            <button
              key={item.id}
              type="button"
              onClick={() => {
                onChange?.(item.id);
                setOpen(false);
              }}
              className={cn(
                "flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left text-sm font-bold transition-all",
                active
                  ? "bg-indigo-600 text-white shadow-glow"
                  : "text-slate-600 hover:bg-indigo-50 hover:text-indigo-700 dark:text-slate-300 dark:hover:bg-slate-900 dark:hover:text-white",
              )}
            >
              <Icon className="h-4 w-4" />
              <span>{item.label}</span>
            </button>
          );
      })}
    </nav>
  );

  return (
    <>
      <div className="lg:hidden">
        <button
          type="button"
          onClick={() => setOpen(true)}
          aria-label="Open navigation menu"
          className="inline-flex h-10 items-center gap-2 rounded-xl border border-slate-200 bg-white px-3 text-sm font-bold text-slate-700 shadow-sm dark:border-slate-800 dark:bg-slate-950 dark:text-slate-100"
        >
          <Menu className="h-4 w-4" />
          Menu
        </button>
      </div>

      <aside className="sticky top-24 hidden h-[calc(100vh-7rem)] w-72 shrink-0 rounded-2xl border border-slate-200 bg-white p-4 shadow-sm dark:border-slate-800 dark:bg-slate-950 lg:block">
        <div className="mb-5 rounded-xl border border-slate-200 bg-slate-50 p-4 dark:border-slate-800 dark:bg-slate-900">
          <p className="text-sm font-black text-slate-950 dark:text-white">{title}</p>
          <p className="mt-1 text-xs leading-relaxed text-slate-500 dark:text-slate-400">{subtitle}</p>
        </div>
        <div>{nav}</div>
      </aside>

      {open ? (
        <div className="fixed inset-0 z-50 bg-slate-950/35 backdrop-blur-sm lg:hidden">
          <div className="h-full w-[min(20rem,86vw)] bg-white p-4 shadow-2xl">
            <div className="mb-5 flex items-start justify-between gap-4">
              <div>
                <p className="text-sm font-black text-slate-950">{title}</p>
                <p className="mt-1 text-xs text-slate-500">{subtitle}</p>
              </div>
              <button
                type="button"
                onClick={() => setOpen(false)}
                aria-label="Close navigation menu"
                className="rounded-lg p-2 text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-900"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            {nav}
          </div>
        </div>
      ) : null}
    </>
  );
}
