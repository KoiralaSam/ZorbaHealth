import type { LucideIcon } from "lucide-react";
import { cn } from "../../lib/utils";

type StatCardProps = {
  icon: LucideIcon;
  label: string;
  value: string | number;
  trend?: string;
  tone?: "indigo" | "orange" | "emerald" | "rose" | "slate";
};

const toneMap = {
  indigo: "bg-indigo-50 text-indigo-600 ring-indigo-100",
  orange: "bg-orange-50 text-orange-600 ring-orange-100",
  emerald: "bg-emerald-50 text-emerald-600 ring-emerald-100",
  rose: "bg-rose-50 text-rose-600 ring-rose-100",
  slate: "bg-slate-100 text-slate-600 ring-slate-200",
};

export function StatCard({
  icon: Icon,
  label,
  value,
  trend,
  tone = "indigo",
}: StatCardProps) {
  return (
    <div className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm dark:border-slate-800 dark:bg-slate-950">
      <div className="flex items-start justify-between gap-4">
        <div>
          <p className="text-[11px] font-black uppercase tracking-[0.18em] text-slate-500">
            {label}
          </p>
          <div className="mt-3 text-3xl font-black tracking-tight text-slate-950 dark:text-slate-50">
            {value}
          </div>
        </div>
        <div
          className={cn(
            "flex h-11 w-11 items-center justify-center rounded-xl ring-1",
            toneMap[tone],
          )}
        >
          <Icon className="h-5 w-5" />
        </div>
      </div>
      {trend ? (
        <p className="mt-4 border-t border-slate-100 pt-3 text-xs font-semibold text-slate-500 dark:border-slate-800 dark:text-slate-400">
          {trend}
        </p>
      ) : null}
    </div>
  );
}
