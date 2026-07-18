import * as React from "react";
import { cn } from "../../lib/utils";

export interface InputProps
  extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  icon?: React.ReactNode;
  error?: string;
}

const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ className, label, icon, error, id, ...props }, ref) => {
    const generatedId = React.useId();
    const inputId = id ?? generatedId;

    return (
      <div className="space-y-2">
        <label
          htmlFor={inputId}
          className="text-xs font-bold uppercase tracking-wide text-slate-600"
        >
          {label}
        </label>
        <div className="relative">
          {icon ? (
            <span className="pointer-events-none absolute left-3 top-1/2 flex -translate-y-1/2 text-slate-400">
              {icon}
            </span>
          ) : null}
          <input
            id={inputId}
            ref={ref}
            className={cn(
              "h-12 w-full rounded-xl border border-slate-200 bg-white px-4 text-sm text-slate-950 shadow-sm outline-none transition-all placeholder:text-slate-400 focus:border-indigo-400 focus:ring-4 focus:ring-indigo-100",
              icon && "pl-10",
              error &&
                "border-rose-300 bg-rose-50/40 focus:border-rose-400 focus:ring-rose-100",
              className,
            )}
            {...props}
          />
        </div>
        {error ? <p className="text-xs font-medium text-rose-600">{error}</p> : null}
      </div>
    );
  },
);
Input.displayName = "Input";

export { Input };
