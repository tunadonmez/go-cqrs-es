import { Card } from "@/components/ui/card";
import { formatMetricKey } from "@/lib/format";
import { MetricsSnapshot } from "@/lib/types";

export function MetricsGrid({
  title,
  metrics
}: {
  title: string;
  metrics?: MetricsSnapshot | null;
}) {
  const entries = Object.entries(metrics ?? {}).slice(0, 12);

  return (
    <Card>
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-slate-950">{title}</h3>
        <span className="text-xs uppercase tracking-[0.2em] text-slate-400">JSON snapshot</span>
      </div>
      {entries.length === 0 ? (
        <p className="mt-4 text-sm text-slate-500">No metrics returned.</p>
      ) : (
        <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {entries.map(([key, value]) => (
            <div key={key} className="rounded-2xl border border-slate-200 bg-slate-50 p-4">
              <p className="text-xs uppercase tracking-[0.16em] text-slate-500">
                {formatMetricKey(key)}
              </p>
              <p className="mt-2 text-2xl font-semibold text-slate-950">{value}</p>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}
