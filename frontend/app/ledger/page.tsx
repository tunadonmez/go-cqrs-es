"use client";

import { useState } from "react";

import { LedgerEntriesTable } from "@/components/ledger/ledger-entries-table";
import { Card } from "@/components/ui/card";
import { EmptyState, ErrorState, LoadingState } from "@/components/ui/state";
import { useLedgerEntries } from "@/hooks/use-query-data";

export default function LedgerPage() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [sortBy, setSortBy] = useState("occurredAt");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [walletId, setWalletId] = useState("");
  const [entryType, setEntryType] = useState("");
  const [eventType, setEventType] = useState("");
  const [occurredFrom, setOccurredFrom] = useState("");
  const [occurredTo, setOccurredTo] = useState("");

  const ledger = useLedgerEntries({
    page,
    pageSize,
    sortBy,
    sortOrder,
    walletId,
    entryType,
    eventType,
    occurredFrom,
    occurredTo
  });

  if (ledger.isLoading) {
    return <LoadingState label="Loading ledger entries..." />;
  }

  if (ledger.error) {
    return (
      <ErrorState
        title="Could not load ledger entries"
        body={ledger.error instanceof Error ? ledger.error.message : "Unknown ledger query failure."}
      />
    );
  }

  const items = ledger.data?.ledgerEntries ?? [];

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
            placeholder="walletId / aggregateId"
          />
          <select
            value={entryType}
            onChange={(event) => {
              setPage(1);
              setEntryType(event.target.value);
            }}
            className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
          >
            <option value="">All entry types</option>
            <option value="DEBIT">DEBIT</option>
            <option value="CREDIT">CREDIT</option>
          </select>
          <input
            value={eventType}
            onChange={(event) => {
              setPage(1);
              setEventType(event.target.value);
            }}
            className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
            placeholder="eventType"
          />
          <div className="grid grid-cols-2 gap-4">
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
        </div>
        <div className="mt-4 flex flex-wrap gap-3">
          <select value={sortBy} onChange={(event) => setSortBy(event.target.value)} className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm">
            <option value="occurredAt">Sort by occurredAt</option>
            <option value="createdAt">Sort by createdAt</option>
          </select>
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
      </Card>

      {items.length === 0 ? (
        <EmptyState
          title="No ledger entries for the current filter set"
          body="Ledger rows are projected from read-side events. A replay may be required after upgrading an existing environment."
        />
      ) : (
        <Card>
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-xl font-semibold text-slate-950">Ledger Read Model</h2>
              <p className="mt-1 text-sm text-slate-500">
                Debit/credit entries projected from wallet events for audit-oriented inspection.
              </p>
            </div>
            <div className="text-sm text-slate-500">
              Returned: {ledger.data?.pagination?.returnedItems ?? items.length}
            </div>
          </div>
          <div className="mt-6">
            <LedgerEntriesTable entries={items} />
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
            if (ledger.data?.pagination?.hasMore) {
              setPage((current) => current + 1);
            }
          }}
          disabled={!ledger.data?.pagination?.hasMore}
          className="rounded-2xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 disabled:opacity-50"
        >
          Next
        </button>
      </div>
    </div>
  );
}
