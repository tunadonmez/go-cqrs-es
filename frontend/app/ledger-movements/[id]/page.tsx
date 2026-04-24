"use client";

import Link from "next/link";
import { useParams } from "next/navigation";

import { LedgerEntriesTable } from "@/components/ledger/ledger-entries-table";
import { Card } from "@/components/ui/card";
import { EmptyState, ErrorState, LoadingState } from "@/components/ui/state";
import { StatusBadge } from "@/components/ui/status-badge";
import { useLedgerMovement, useLedgerEntries } from "@/hooks/use-query-data";
import { formatCurrency, formatDateTime, truncateMiddle } from "@/lib/format";

function movementTone(status: string) {
  switch (status) {
  case "COMPLETED":
    return "ready";
  case "PENDING":
    return "pending";
  default:
    return "down";
  }
}

export default function LedgerMovementDetailPage() {
  const params = useParams<{ id: string }>();
  const movementId = params.id;
  const movement = useLedgerMovement(movementId);
  const entries = useLedgerEntries({
    page: 1,
    pageSize: 50,
    sortBy: "occurredAt",
    sortOrder: "asc",
    movementId
  });

  if (movement.isLoading || entries.isLoading) {
    return <LoadingState label="Loading ledger movement..." />;
  }

  if (movement.error || entries.error) {
    return (
      <ErrorState
        title="Could not load ledger movement"
        body={
          movement.error instanceof Error
            ? movement.error.message
            : entries.error instanceof Error
              ? entries.error.message
              : "Unknown ledger movement failure."
        }
      />
    );
  }

  const item = movement.data?.ledgerMovement;
  if (!item) {
    return (
      <EmptyState
        title="Ledger movement not found"
        body="The journal row was not returned by the query API. A replay may still be required in this environment."
      />
    );
  }

  const movementEntries = entries.data?.ledgerEntries ?? [];

  return (
    <div className="space-y-6">
      <Card>
        <div className="flex items-start justify-between gap-6">
          <div>
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-teal-700">Ledger Movement</p>
            <h1 className="mt-2 text-2xl font-semibold text-slate-950">{item.movementType}</h1>
            <p className="mt-2 text-sm text-slate-500">
              {truncateMiddle(item.id)} {item.reference ? `· ${item.reference}` : ""}
            </p>
          </div>
          <StatusBadge label={item.status} tone={movementTone(item.status)} />
        </div>

        <div className="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <Metric label="Occurred" value={formatDateTime(item.occurredAt)} />
          <Metric label="Debit" value={formatCurrency(item.totalDebit, item.currency || "USD")} />
          <Metric label="Credit" value={formatCurrency(item.totalCredit, item.currency || "USD")} />
          <Metric label="Entry Count" value={String(item.entryCount)} />
          <Metric label="Source Wallet" value={item.sourceWalletId ? truncateMiddle(item.sourceWalletId) : "external"} />
          <Metric label="Destination Wallet" value={item.destinationWalletId ? truncateMiddle(item.destinationWalletId) : "external"} />
          <Metric label="Event Type" value={item.eventType || "n/a"} />
          <Metric label="Event ID" value={item.eventId ? truncateMiddle(item.eventId) : "n/a"} />
        </div>

        <div className="mt-6 flex flex-wrap gap-3 text-sm">
          <Link href={`/ledger?movementId=${item.id}`} className="rounded-2xl border border-slate-200 px-4 py-2 font-medium text-slate-700">
            Open in ledger entries view
          </Link>
          {item.sourceWalletId ? (
            <Link href={`/wallets/${item.sourceWalletId}`} className="rounded-2xl border border-slate-200 px-4 py-2 font-medium text-slate-700">
              Source wallet
            </Link>
          ) : null}
          {item.destinationWalletId ? (
            <Link href={`/wallets/${item.destinationWalletId}`} className="rounded-2xl border border-slate-200 px-4 py-2 font-medium text-slate-700">
              Destination wallet
            </Link>
          ) : null}
        </div>
      </Card>

      <Card>
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-slate-950">Linked Ledger Entries</h2>
            <p className="mt-1 text-sm text-slate-500">
              These rows share the same explicit `movementId` and are what the movement totals are derived from.
            </p>
          </div>
          <div className="text-sm text-slate-500">Returned: {movementEntries.length}</div>
        </div>
        {movementEntries.length === 0 ? (
          <p className="mt-6 text-sm text-slate-500">No ledger entries were returned for this movement.</p>
        ) : (
          <div className="mt-6">
            <LedgerEntriesTable entries={movementEntries} fallbackCurrency={item.currency} />
          </div>
        )}
      </Card>
    </div>
  );
}

function Metric({
  label,
  value
}: {
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-3xl border border-slate-200 bg-slate-50/70 p-4">
      <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">{label}</p>
      <p className="mt-2 text-sm text-slate-950">{value}</p>
    </div>
  );
}
