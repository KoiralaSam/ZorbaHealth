import { AlertTriangle, CheckCircle2, Info } from "lucide-react";
import { cn } from "../lib/utils";

type StatusBannerProps = {
  tone: "error" | "success" | "info";
  message: string;
  className?: string;
};

const tones = {
  error: {
    icon: AlertTriangle,
    className:
      "border-rose-200 bg-rose-50/80 text-rose-700 dark:border-rose-900 dark:bg-rose-950/60 dark:text-rose-200",
  },
  success: {
    icon: CheckCircle2,
    className:
      "border-emerald-200 bg-emerald-50/80 text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950/60 dark:text-emerald-200",
  },
  info: {
    icon: Info,
    className:
      "border-indigo-200 bg-indigo-50/80 text-indigo-700 dark:border-indigo-900 dark:bg-indigo-950/60 dark:text-indigo-200",
  },
};

export function StatusBanner({ tone, message, className }: StatusBannerProps) {
  const Icon = tones[tone].icon;

  return (
    <div
      className={cn(
        "flex items-start gap-3 rounded-2xl border p-4 text-sm",
        tones[tone].className,
        className,
      )}
    >
      <Icon className="mt-0.5 h-5 w-5 shrink-0" />
      <span>{message}</span>
    </div>
  );
}
