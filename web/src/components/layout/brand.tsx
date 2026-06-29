"use client";

import Image from "next/image";
import Link from "next/link";
import { cn } from "../../lib/utils";

type BrandProps = {
  href?: string;
  compact?: boolean;
  className?: string;
  markClassName?: string;
  wordmarkClassName?: string;
};

export function Brand({
  href = "/",
  compact = false,
  className,
  markClassName,
  wordmarkClassName,
}: BrandProps) {
  const content = (
    <div className={cn("flex items-center gap-3", className)}>
      <div
        className={cn(
          "flex h-11 w-11 items-center justify-center overflow-hidden rounded-2xl bg-white/80 shadow-glow ring-1 ring-white/70",
          compact && "h-10 w-10",
          markClassName,
        )}
      >
        <Image
          src="/brand/zorba-mark.png"
          alt="Zorba Health"
          width={44}
          height={44}
          className="h-full w-full object-cover"
          priority
        />
      </div>
      <div className={cn("flex flex-col", wordmarkClassName)}>
        <span className="gradient-text text-lg font-black tracking-tight sm:text-xl">
          Zorba Health
        </span>
        {!compact ? (
          <span className="text-[10px] font-bold uppercase tracking-[0.24em] text-slate-500 dark:text-slate-400">
            Voice Care Platform
          </span>
        ) : null}
      </div>
    </div>
  );

  return (
    <Link href={href} className="inline-flex items-center focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-4">
      {content}
    </Link>
  );
}
