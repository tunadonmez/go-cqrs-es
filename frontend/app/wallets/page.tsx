"use client";

import { useState } from "react";

import { WalletsTable } from "@/components/wallets/wallets-table";
import { Card } from "@/components/ui/card";
import { EmptyState, ErrorState, LoadingState } from "@/components/ui/state";
import { useWallets } from "@/hooks/use-query-data";

export default function WalletsPage() {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [sortBy, setSortBy] = useState("createdAt");
  const [sortOrder, setSortOrder] = useState<"asc" | "desc">("desc");
  const [currency, setCurrency] = useState("");

  const wallets = useWallets({ page, pageSize, sortBy, sortOrder, currency });

  if (wallets.isLoading) {
    return <LoadingState label="Loading wallets..." />;
  }

  if (wallets.error) {
    return <ErrorState title="Could not load wallets" body={wallets.error instanceof Error ? wallets.error.message : "Unknown wallet error."} />;
  }

  const items = wallets.data?.wallets ?? [];
  if (items.length === 0) {
    return (
      <EmptyState
        title="No wallets in the query model"
        body="The query service returned no wallets. This can mean no commands have been processed yet or the read side has not caught up."
      />
    );
  }

  return (
    <div className="space-y-6">
      <Card>
        <div className="flex flex-col gap-4 lg:flex-row lg:items-end">
          <label className="flex-1">
            <span className="mb-2 block text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">
              Currency Filter
            </span>
            <input
              value={currency}
              onChange={(event) => {
                setPage(1);
                setCurrency(event.target.value.toUpperCase());
              }}
              className="w-full rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
              placeholder="USD"
            />
          </label>
          <label>
            <span className="mb-2 block text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">
              Sort By
            </span>
            <select value={sortBy} onChange={(event) => setSortBy(event.target.value)} className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm">
              <option value="createdAt">Created</option>
              <option value="balance">Balance</option>
              <option value="owner">Owner</option>
            </select>
          </label>
          <label>
            <span className="mb-2 block text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">
              Order
            </span>
            <select value={sortOrder} onChange={(event) => setSortOrder(event.target.value as "asc" | "desc")} className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm">
              <option value="asc">Ascending</option>
              <option value="desc">Descending</option>
            </select>
          </label>
          <label>
            <span className="mb-2 block text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">
              Page Size
            </span>
            <select value={pageSize} onChange={(event) => setPageSize(Number(event.target.value))} className="rounded-2xl border border-slate-200 bg-slate-50 px-4 py-3 text-sm">
              <option value="10">10</option>
              <option value="20">20</option>
              <option value="50">50</option>
            </select>
          </label>
        </div>
      </Card>

      <Card>
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-slate-950">Wallet Read Model</h2>
            <p className="mt-1 text-sm text-slate-500">Data fetched from `wallet-query`, not the write-side source of truth.</p>
          </div>
          <div className="text-sm text-slate-500">
            Returned: {wallets.data?.pagination?.returnedItems ?? items.length}
          </div>
        </div>
        <div className="mt-6">
          <WalletsTable wallets={items} />
        </div>
        <div className="mt-6 flex items-center justify-between">
          <button
            onClick={() => setPage((current) => Math.max(1, current - 1))}
            className="rounded-2xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700"
          >
            Previous
          </button>
          <p className="text-sm text-slate-500">Page {page}</p>
          <button
            onClick={() => {
              if (wallets.data?.pagination?.hasMore) {
                setPage((current) => current + 1);
              }
            }}
            className="rounded-2xl border border-slate-200 px-4 py-2 text-sm font-medium text-slate-700 disabled:opacity-50"
            disabled={!wallets.data?.pagination?.hasMore}
          >
            Next
          </button>
        </div>
      </Card>
    </div>
  );
}
