"use client";

import type { SidebarItem } from "../ui/sidebar";
import { Sidebar } from "../ui/sidebar";
import { AppHeader } from "./app-header";
import { Button } from "../ui/button";
import { Brand } from "./brand";
import { LogOut } from "lucide-react";

type DashboardShellProps = {
  title: string;
  subtitle: string;
  navItems: SidebarItem[];
  activeSection: string;
  onSectionChange: (section: string) => void;
  onLogout: () => void;
  headerLinks?: { href: string; label: string }[];
  children: React.ReactNode;
};

export function DashboardShell({
  title,
  subtitle,
  navItems,
  activeSection,
  onSectionChange,
  onLogout,
  headerLinks,
  children,
}: DashboardShellProps) {
  return (
    <main className="relative min-h-screen bg-slate-100 pb-12 text-slate-950 dark:bg-slate-950 dark:text-slate-50">
      <AppHeader
        links={headerLinks}
        className="border-slate-200 bg-white/95 text-slate-950 shadow-sm dark:border-slate-800 dark:bg-slate-950/95 dark:text-white"
        rightSlot={
          <div className="hidden items-center gap-3 md:flex">
            <Brand compact href="/" />
            <Button type="button" variant="outline" onClick={onLogout}>
              <LogOut className="h-4 w-4" />
              Sign out
            </Button>
          </div>
        }
      />
      <div className="mx-auto flex max-w-7xl flex-col gap-6 px-5 pt-6 lg:flex-row">
        <Sidebar
          title={title}
          subtitle={subtitle}
          items={navItems}
          activeItem={activeSection}
          onChange={onSectionChange}
        />
        <div className="min-w-0 flex-1 space-y-6">{children}</div>
      </div>
    </main>
  );
}
