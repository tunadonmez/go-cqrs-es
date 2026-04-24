"use client";

import Link from "next/link";

import { MetricsGrid } from "@/components/metrics/metrics-grid";
import { Card } from "@/components/ui/card";
import { ErrorState, LoadingState } from "@/components/ui/state";
import { StatusBadge } from "@/components/ui/status-badge";
import { useCommandHealth, useCommandMetrics, useCommandReady } from "@/hooks/use-command-data";
import { useQueryHealth, useQueryMetrics, useQueryReady } from "@/hooks/use-query-data";

function ServiceCard({
  title,
  health,
  ready
}: {
  title: string;
  health?: { status: string } | null;
  ready?: { status: string; reason?: string } | null;
}) {
  return (
    <Card>
      <h3 className="text-lg font-semibold text-slate-950">{title}</h3>
      <div className="mt-4 flex flex-wrap gap-2">
        <StatusBadge label={`health: ${health?.status ?? "unknown"}`} tone={health?.status} />
        <StatusBadge label={`ready: ${ready?.status ?? "unknown"}`} tone={ready?.status} />
      </div>
      {ready?.reason ? <p className="mt-3 text-sm text-rose-700">Reason: {ready.reason}</p> : null}
    </Card>
  );
}

export default function DashboardPage() {
  const commandHealth = useCommandHealth();
  const commandReady = useCommandReady();
  const commandMetrics = useCommandMetrics();
  const queryHealth = useQueryHealth();
  const queryReady = useQueryReady();
  const queryMetrics = useQueryMetrics();

  if (commandHealth.isLoading || queryHealth.isLoading) {
    return <LoadingState label="Loading service health..." />;
  }

  if (commandHealth.error || queryHealth.error) {
    return (
      <ErrorState
        title="Could not load dashboard status"
        body={
          commandHealth.error instanceof Error
            ? commandHealth.error.message
            : queryHealth.error instanceof Error
              ? queryHealth.error.message
              : "Unknown dashboard failure."
        }
      />
    );
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-6 lg:grid-cols-[1.3fr_0.7fr]">
        <Card className="overflow-hidden">
          <div className="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.26em] text-teal-700">
                CQRS + Event Sourcing
              </p>
              <h2 className="mt-3 text-4xl font-semibold leading-tight text-slate-950">
                Observe the write side, the read side, and the lag between them.
              </h2>
              <p className="mt-4 max-w-3xl text-sm leading-6 text-slate-600">
                This console is intentionally split by responsibility: commands go to the write service,
                read models come from the query service, and dead letters remain explicit instead of being
                hidden behind retries.
              </p>
            </div>
            <div className="rounded-3xl bg-slate-950 p-5 text-sm text-slate-200">
              <p>HTTP command</p>
              <p className="mt-2">MongoDB event log + outbox</p>
              <p className="mt-2">Kafka delivery</p>
              <p className="mt-2">PostgreSQL projection</p>
            </div>
          </div>
        </Card>
        <Card>
          <h3 className="text-lg font-semibold text-slate-950">Quick Links</h3>
          <div className="mt-4 grid gap-3">
            {[
              { href: "/wallets", label: "Inspect read-model wallets" },
              { href: "/commands", label: "Issue command-side actions" },
              { href: "/dead-letters", label: "Inspect and reprocess dead letters" },
              { href: "/operations", label: "Operational replay and repair notes" }
            ].map((item) => (
              <Link
                key={item.href}
                href={item.href}
                className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-4 text-sm font-medium text-slate-800 hover:bg-slate-100"
              >
                {item.label}
              </Link>
            ))}
          </div>
        </Card>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <ServiceCard title="Command Service" health={commandHealth.data ?? undefined} ready={commandReady.data ?? undefined} />
        <ServiceCard title="Query Service" health={queryHealth.data ?? undefined} ready={queryReady.data ?? undefined} />
      </div>

      <div className="grid gap-6 xl:grid-cols-2">
        <MetricsGrid title="Command Metrics" metrics={commandMetrics.data} />
        <MetricsGrid title="Query Metrics" metrics={queryMetrics.data} />
      </div>
    </div>
  );
}
