import type { LucideIcon } from "lucide-react";
import { Button } from "./ui/button";

type EmptyStateProps = {
  icon: LucideIcon;
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
};

export function EmptyState({
  icon: Icon,
  title,
  description,
  actionLabel,
  onAction,
}: EmptyStateProps) {
  return (
    <div className="clinical-card flex flex-col items-center rounded-3xl px-6 py-10 text-center dark:border-slate-800 dark:bg-slate-950">
      <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-indigo-50 text-indigo-600 ring-1 ring-indigo-100 dark:bg-indigo-950/80 dark:text-indigo-200 dark:ring-indigo-900">
        <Icon className="h-6 w-6" />
      </div>
      <h3 className="mt-5 text-xl font-black tracking-tight text-slate-950 dark:text-slate-50">
        {title}
      </h3>
      <p className="mt-2 max-w-md text-sm leading-6 text-slate-500 dark:text-slate-400">
        {description}
      </p>
      {actionLabel && onAction ? (
        <Button type="button" variant="outline" className="mt-5" onClick={onAction}>
          {actionLabel}
        </Button>
      ) : null}
    </div>
  );
}
