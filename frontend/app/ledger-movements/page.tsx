"use client";

import { useState } from "react";

import { LedgerMovementsTable } from "@/components/ledger/ledger-movements-table";
import { Card } from "@/components/ui/card";
import { EmptyState, ErrorState, LoadingState } from "@/components/ui/state";
import { useLedgerMovements } from "@/hooks/use-query-data";

export default function LedgerMovementsPage() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [sortBy, setSortBy] = useState("occurredAt");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [walletId, setWalletId] = useState("");
  const [movementType, setMovementType] = useState("");
  const [status, setStatus] = useState("");
  const [reference, setReference] = useState("");
  const [occurredFrom, setOccurredFrom] = useState("");
  const [occurredTo, setOccurredTo] = useState("");

  const movements = useLedgerMovements({
    page,
    pageSize,
    sortBy,
    sortOrder,
    walletId,
    movementType,
    status,
    reference,
    occurredFrom,
    occurredTo
  });

  if (movements.isLoading) {
    return <LoadingState label="Loading ledger movements..." />;
  }

  if (movements.error) {
    return (
      <ErrorState
        title="Could not load ledger movements"
        body={movements.error instanceof Error ? movements.error.message : "Unknown movement query failure."}
      />
    );
  }

  const items = movements.data?.ledgerMovements ?? [];

  return (
    <div className="space-y-6">
      <Card>
        <div className="grid gap-4 lg:grid-cols-4">
          <input
            value={walletId}
            onChange={(event) => {
              setPage(1);
              setWalletId(event.target.value);
            }}
            className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
            placeholder="walletId"
          />
          <select
            value={movementType}
            onChange={(event) => {
              setPage(1);
              setMovementType(event.target.value);
            }}
            className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
          >
            <option value="">All movement types</option>
            <option value="OPENING_BALANCE">OPENING_BALANCE</option>
            <option value="CREDIT">CREDIT</option>
            <option value="DEBIT">DEBIT</option>
            <option value="TRANSFER">TRANSFER</option>
          </select>
          <select
            value={status}
            onChange={(event) => {
              setPage(1);
              setStatus(event.target.value);
            }}
            className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
          >
            <option value="">All statuses</option>
            <option value="COMPLETED">COMPLETED</option>
            <option value="PENDING">PENDING</option>
            <option value="INCONSISTENT">INCONSISTENT</option>
          </select>
          <input
            value={reference}
            onChange={(event) => {
              setPage(1);
              setReference(event.target.value);
            }}
            className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
            placeholder="reference"
          />
        </div>
        <div className="mt-4 grid gap-4 lg:grid-cols-4">
          <div className="grid grid-cols-2 gap-4 lg:col-span-2">
            <input
              value={occurredFrom}
              onChange={(event) => {
                setPage(1);
                setOccurredFrom(event.target.value);
              }}
              type="date"
              className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
            />
            <input
              value={occurredTo}
              onChange={(event) => {
                setPage(1);
                setOccurredTo(event.target.value);
              }}
              type="date"
              className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
            />
          </div>
          <select value={sortBy} onChange={(event) => setSortBy(event.target.value)} className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm">
            <option value="occurredAt">Sort by occurredAt</option>
            <option value="createdAt">Sort by createdAt</option>
          </select>
          <div className="grid grid-cols-2 gap-4">
            <select value={sortOrder} onChange={(event) => setSortOrder(event.target.value as "asc" | "desc")} className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm">
              <option value="desc">Descending</option>
              <option value="asc">Ascending</option>
            </select>
            <select value={pageSize} onChange={(event) => setPageSize(Number(event.target.value))} className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm">
              <option value="10">10 rows</option>
              <option value="20">20 rows</option>
              <option value="50">50 rows</option>
            </select>
          </div>
        </div>
      </Card>

      {items.length === 0 ? (
        <EmptyState
          title="No ledger movements for the current filter set"
          body="Movement rows are explicit projected journal summaries. Existing environments need a replay to populate them."
        />
      ) : (
        <Card>
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-xl font-semibold text-slate-950">Ledger Movements</h2>
              <p className="mt-1 text-sm text-slate-500">
                First-class journal rows projected alongside ledger entries for audit and operator workflows.
              </p>
            </div>
            <div className="text-sm text-slate-500">
              Returned: {movements.data?.pagination?.returnedItems ?? items.length}
            </div>
          </div>
          <div className="mt-6">
            <LedgerMovementsTable movements={items} />
          </div>
        </Card>
      )}

      <div className="flex items-center justify-between">
        <button onClick={() => setPage((current) => Math.max(1, current - 1))} className="rounded-2xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700">
          Previous
        </button>
        <p className="text-sm text-slate-500">Page {page}</p>
        <button
          onClick={() => {
            if (movements.data?.pagination?.hasMore) {
              setPage((current) => current + 1);
            }
          }}
          disabled={!movements.data?.pagination?.hasMore}
          className="rounded-2xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 disabled:opacity-50"
        >
          Next
        </button>
      </div>
    </div>
  );
}
