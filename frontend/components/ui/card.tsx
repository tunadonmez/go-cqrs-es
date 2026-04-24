import type { ReactNode } from "react";
import { clsx } from "clsx";

export function Card({
  children,
  className
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <section
      className={clsx(
        "rounded-3xl border border-slate-200/70 bg-white/90 p-6 shadow-panel backdrop-blur",
        className
      )}
    >
      {children}
    </section>
  );
}
