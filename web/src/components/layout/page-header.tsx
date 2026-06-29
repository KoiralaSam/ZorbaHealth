import { Badge } from "../ui/badge";

type PageHeaderProps = {
  eyebrow: string;
  title: string;
  description: string;
  actions?: React.ReactNode;
};

export function PageHeader({ eyebrow, title, description, actions }: PageHeaderProps) {
  return (
    <div className="flex flex-col gap-4 rounded-2xl border border-slate-200 bg-white p-6 shadow-sm dark:border-slate-800 dark:bg-slate-950 md:flex-row md:items-end md:justify-between">
      <div>
        <Badge className="rounded-full border border-indigo-200/70 bg-indigo-50/80 px-3 py-1 text-indigo-700 dark:border-white/10 dark:bg-white/10 dark:text-indigo-200">
          {eyebrow}
        </Badge>
        <h1 className="mt-4 text-3xl font-black tracking-tight text-slate-950 dark:text-slate-50 md:text-4xl">
          {title}
        </h1>
        <p className="mt-2 max-w-3xl text-sm leading-7 text-slate-500 dark:text-slate-400">
          {description}
        </p>
      </div>
      {actions ? <div className="flex flex-wrap items-center gap-3">{actions}</div> : null}
    </div>
  );
}
