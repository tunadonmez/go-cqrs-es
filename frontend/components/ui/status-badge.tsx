import { clsx } from "clsx";

const toneMap: Record<string, string> = {
  up: "bg-emerald-100 text-emerald-800",
  ready: "bg-emerald-100 text-emerald-800",
  down: "bg-rose-100 text-rose-800",
  pending: "bg-amber-100 text-amber-900",
  resolved: "bg-teal-100 text-teal-900",
  permanent: "bg-rose-100 text-rose-800",
  retries_exhausted: "bg-orange-100 text-orange-800"
};

export function StatusBadge({
  label,
  tone
}: {
  label: string;
  tone?: string;
}) {
  const key = (tone ?? label).toLowerCase();
  return (
    <span
      className={clsx(
        "inline-flex rounded-full px-2.5 py-1 text-xs font-semibold uppercase tracking-[0.18em]",
        toneMap[key] ?? "bg-slate-200 text-slate-700"
      )}
    >
      {label}
    </span>
  );
}
